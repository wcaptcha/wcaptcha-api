package api

import (
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"wcaptcha/store"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const (
	RSA_KEY_SIZE = 512
	RSA_KEY_TTL  = 600
)

// var S3 *s3kv.Storage
var Store store.Storer

type Site struct {
	SecretKey string
	APIKey    string

	RSAKey              *rsa.PrivateKey
	OldRSAKey           *rsa.PrivateKey
	RSAKeyCreateTime    int64
	OldRSAKeyCreateTime int64

	// RSAKey 的总计轮换次数（总共重新生成了多少次 RSAKey）
	RSAKeyRegenerateCount int

	// 难度，客户端需要计算多少次平方取模，在 2020 年的消费级 CPU 上，Hardness = 2**20 时大约需要 100ms 的时间可计算出结果
	Hardness int

	CreateTime       int64
	CreatorUserAgent string
	HMACKey          []byte
}

func NewSite() *Site {
	// 1. 生成站点 KEY 和 SECRET
	rand.Seed(time.Now().Unix())
	api_secret_buf := make([]byte, 32)

	_, err := rand.Read(api_secret_buf)
	if err != nil {
		log.Printf("无法创建随机数: %v", err)
		return nil
	}

	api_key_buf := sha256.Sum256(api_secret_buf)

	api_key_b64 := base64.RawURLEncoding.EncodeToString(api_key_buf[:])
	api_secret_b64 := base64.RawURLEncoding.EncodeToString(api_secret_buf)

	rsa_key, err := rsa.GenerateKey(crand.Reader, RSA_KEY_SIZE)
	if err != nil {
		log.Printf("无法生成 RSA 密钥对: %v", err)
		return nil
	}

	s := Site{
		APIKey:     api_key_b64,
		SecretKey:  api_secret_b64,
		RSAKey:     rsa_key,
		CreateTime: time.Now().Unix(),
		Hardness:   1<<22 - 1,
	}
	s.HMACKey = make([]byte, 16)
	rand.Read(s.HMACKey)

	return &s
}

// 视情况更新一个站点的密钥
func (s *Site) UpdateKeyIfNeeded() bool {
	isUpdated := false
	var err error

	ts := time.Now().Unix()

	if ts-s.RSAKeyCreateTime < RSA_KEY_TTL {
		return false
	} else {
		isUpdated = true

		s.OldRSAKey = s.RSAKey
		s.OldRSAKeyCreateTime = s.RSAKeyCreateTime

		s.RSAKey, err = rsa.GenerateKey(crand.Reader, RSA_KEY_SIZE)
		s.RSAKeyCreateTime = ts

		s.RSAKeyRegenerateCount++

		if err != nil {
			log.Printf("严重错误：更新密钥失败，GenerateKey 返回错误: %v", err)
		}
	}

	return isUpdated
}

// 根据 APIKey 获取一个 site 的数据
func siteGet(apiKey string) (*Site, error) {
	var site Site
	err := Store.Get(fmt.Sprintf("site/%s", apiKey), &site)
	return &site, err
}

func InitGin() *gin.Engine {
	var err error

	rand.Seed(time.Now().UnixNano())

	switch os.Getenv("STORAGE") {
	case "s3":
		Store = new(store.S3)
	case "file":
		Store = new(store.File)
	default:
		fmt.Printf("环境变量 `STORAGE' 配置错误或不存在，请确认环境变量已正确配置")
		os.Exit(0)
	}

	err = Store.Init()
	if err != nil {
		log.Printf("无法创建存储连接: %v", err)
		os.Exit(0)
	}

	route := gin.Default()

	route.Use(cors.Default())

	route.GET("/captcha/problem/get", webCaptchaProblem)
	route.POST("/captcha/verify", webCaptchaVerify)
	route.POST("/site/create", webSiteCreate)
	route.POST("/site/read", webSiteRead)
	route.POST("/site/update", webSiteUpdate)

	route.GET("/ping", func(c *gin.Context) {
		// c.String(200, fmt.Sprintf("pong. %v.\nS3_BUCKET=%s\nS3_ENDPOINT=%s\n", time.Now(), os.Getenv("S3_BUCKET"), os.Getenv("S3_ENDPOINT")))
		c.String(200, fmt.Sprintf("pong. %v.\nSTORAGE=%s", time.Now(), os.Getenv("STORAGE")))
	})

	return route
}

func StartWeb() {
	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8090"
	}
	port, err := strconv.Atoi(portStr)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid PORT `%s'", portStr)
		os.Exit(0)
	}

	route := InitGin()
	route.Run(fmt.Sprintf(":%d", port))
}

func Handler(w http.ResponseWriter, r *http.Request) {
	InitGin().ServeHTTP(w, r)
}

func saveSite(s *Site) error {
	return Store.Put(fmt.Sprintf("site/%s", s.APIKey), s)
}

func nonceIsExists(nonce string) bool {
	t := time.Now()
	p := fmt.Sprintf("nonce/%s-%s", t.Format("2006010215"), nonce)
	p2 := fmt.Sprintf("nonce/%s-%s", t.Add(-1*86400*time.Second).Format("2006010215"), nonce)

	exists, err := Store.KeyExists(p)
	if err != nil {
		log.Printf("无法获知 nonce 是否已经存在，认为其不存在: %v", err)
		return false
	}

	exists2, err := Store.KeyExists(p2)
	if err != nil {
		log.Printf("无法获知 nonce 是否已经存在，认为其不存在: %v", err)
		return false
	}

	return exists || exists2
}

func nonceSet(nonce string) {
	p := fmt.Sprintf("nonce/%s-%s", time.Now().Format("2006010215"), nonce)

	err := Store.Put(p, []byte(fmt.Sprintf("%d", time.Now().Unix())))
	if err != nil {
		log.Printf("Unable to set nonce `%v'", nonce)
	} else {
		log.Printf("设置了一个 nonce `%s'", p)
	}
}

// 是否正在执行 nonce 清理的操作。该变量用于避免多个 nonce 清理程序同时运行
var isNonceCleaning bool = false

// 以 prob 的概率，触发清理过期的 nonce 操作
func nonceClean(prob float32) {
	if isNonceCleaning == true {
		log.Printf("当前有另一个 Nonce 清理程序正在进行中，不会重复运行 Nonce 清理程序")
		return
	}
	isNonceCleaning = true
	defer func() {
		isNonceCleaning = false
	}()

	r := rand.Float32()
	if r >= prob {
		return
	}

	log.Printf("执行一次清理 nonce 的操作")

	keys, err := Store.List("nonce/")
	if err != nil {
		log.Printf("清理 nonce 操作失败，无法获取 nonce 列表: %v", err)
		return
	}

	t := time.Now()
	nowPrefix := fmt.Sprintf("nonce/%s", t.Format("2006010215"))
	prevPrefix := fmt.Sprintf("nonce/%s", t.Add(86400*time.Second).Format("2006010215"))
	for _, v := range keys {
		if strings.HasPrefix(v, nowPrefix) || strings.HasPrefix(v, prevPrefix) {
			continue
		}
		log.Printf("删除 nonce `%s'", v)
		Store.Delete(v)
	}

	log.Printf("nonce 清理操作完成")
}

func init() {
	godotenv.Load()
}
