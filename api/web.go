package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"strings"
	"time"
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/greensea/vdf"
)

// 创建一个网站，为其生成 API_KEY 和 SECRET_KEY
func webSiteCreate(c *gin.Context) {
	// 创建网站数据
	s := NewSite()
	if s == nil {
		webError(c, -1, "Unable to create site")
		return
	}

	// 保存到数据库
	s.CreatorUserAgent = c.GetHeader("User-Agent")
	err := saveSite(s)
	if err != nil {
		log.Printf("无法将 Site 保存到数据库: %v", err)
		webError(c, -2, "Unable to save site data")
		return
	}

	// 返回结果
	c.JSON(200, gin.H{
		"code": 0,
		"site": gin.H{
			"api_key":    s.APIKey,
			"api_secret": s.SecretKey,
		},
	})
}

// 读取一个网站的数据，需要提供 API Secret
// 注意！该接口会返回 APISecret，如果不再使用 APISecret 作为查询参数，则接口应该做相应的修改
func webSiteRead(c *gin.Context) {
	var req struct {
		APISecret string `form:"api_secret" binding:"required"`
	}
	err := c.ShouldBind(&req)
	if err != nil {
		webError(c, -1, err.Error())
		return
	}

	// 1. 根据 API Secret 计算 API Key
	apiSecretBuf, err := base64.RawURLEncoding.DecodeString(req.APISecret)
	if err != nil {
		webError(c, -2, "Invalid API Secret: "+err.Error())
		return
	}
	apiKeyBuf := sha256.Sum256(apiSecretBuf)
	apiKey := base64.RawURLEncoding.EncodeToString(apiKeyBuf[:])

	site, err := siteGet(apiKey)
	if err != nil {
		webError(c, -10, "Can't find site: "+err.Error())
		return
	}

	c.JSON(0, gin.H{
		"code": 0,
		"data": gin.H{
			"site": gin.H{
				"api_key":                  site.APIKey,
				"api_secret":               site.SecretKey,
				"hardness":                 site.Hardness,
				"create_time":              site.CreateTime,
				"rsa_key_regenerate_count": site.RSAKeyRegenerateCount,
				"rsa_key_create_time":      site.RSAKeyCreateTime,
			},
		},
	})
}

// 修改一个网站的数据，需要提供 API Secret
func webSiteUpdate(c *gin.Context) {
	var req struct {
		APISecret string `form:"api_secret" binding:"required"`
		Hardness  int    `form:"hardness" binding:"required"`
	}
	err := c.ShouldBind(&req)
	if err != nil {
		webError(c, -1, err.Error())
		return
	}

	// 1. 根据 API Secret 计算 API Key
	apiSecretBuf, err := base64.RawURLEncoding.DecodeString(req.APISecret)
	if err != nil {
		webError(c, -2, "Invalid API Secret: "+err.Error())
		return
	}
	apiKeyBuf := sha256.Sum256(apiSecretBuf)
	apiKey := base64.RawURLEncoding.EncodeToString(apiKeyBuf[:])

	site, err := siteGet(apiKey)
	if err != nil {
		webError(c, -10, "Can't find site: "+err.Error())
		return
	}

	site.Hardness = req.Hardness
	err = saveSite(site)
	if err != nil {
		webError(c, -20, "Can't save site info"+err.Error())
		return
	}

	c.JSON(0, gin.H{
		"code": 0,
		"data": gin.H{
			"site": gin.H{
				"api_key":                  site.APIKey,
				"hardness":                 site.Hardness,
				"create_time":              site.CreateTime,
				"rsa_key_regenerate_count": site.RSAKeyRegenerateCount,
				"rsa_key_create_time":      site.RSAKeyCreateTime,
			},
		},
	})
}

// 生成一个问题
// 我们在服务端生成以下数据：
// x: 随机数
// h: 对 x 的签名，使用 HMAC_SHA256 以及服务器自定义的密钥进行签名
// n: 模数，这个数据不需要生成，直接使用 site 的 RSA 素数生成
// 给客户端返回：
// question = {X: X, H: H, N: N}
func webCaptchaProblem(c *gin.Context) {
	var req struct {
		APIKey string `form:"api_key" binding:"required"`
	}
	err := c.ShouldBind(&req)
	if err != nil {
		webError(c, -1, err.Error())
		return
	}

	// 1. 查询网站是否存在
	var site Site
	err = Store.Get(fmt.Sprintf("site/%s", req.APIKey), &site)
	if err != nil {
		webError(c, -2, fmt.Sprintf("Site not exists. (Internal error: %s)", err.Error()))
		return
	}

	// 2. 检查 RSA 钥匙对是否已经过期，若已经过期，则更新之
	is_updated := site.UpdateKeyIfNeeded()
	if is_updated {
		log.Printf("站点 %s 的 RSA 密钥对已更新，更新数据库数据", site.APIKey)
		err := saveSite(&site)
		if err != nil {
			log.Printf("保存站点数据失败: %v", err)
			webError(c, -10, "Unable to update key")
			return
		}
	}

	// 3. 生成数据并返回
	n := new(big.Int).Set(site.RSAKey.Primes[0])
	n = n.Mul(n, site.RSAKey.Primes[1])
	x := big.NewInt(rand.Int63())

	h := hmac.New(sha256.New, site.HMACKey).Sum([]byte(x.Text(16)))

	// nB64 := base64.RawStdEncoding.EncodeToString(n.Bytes())
	// xB64 := base64.RawStdEncoding.EncodeToString([]byte(x))
	hB64 := base64.RawURLEncoding.EncodeToString(h)

	//question := fmt.Sprintf("%s.%s.%s", nB64, xB64, hB64)

	// 以 1% 的概率去触发 Nonce 清除操作
	go nonceCleanup(nonceCleanupProb)

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"question": gin.H{
				"x": x.Text(16),
				"h": hB64,
				"n": n.Text(16),
				"t": site.Hardness,
			},
		},
	})
}

// 检查客户端计算出的结果是否正确
// 客户端应提交 prove 和 api_key 参数。其中 prove 数据格式如下：
// X.Y.H
// 其中 X 是此前服务器返回的 x 的原始内容；Y 是计算结果，为小写的十六进制表达式；H 为此前服务器返回的签名原始内容；
// <del>N 为模数，为小写的十六进制格式</del>
func webCaptchaVerify(c *gin.Context) {
	var verifyElapsed time.Duration

	// 1. 解析客户端数据
	var req struct {
		Prove  string `form:"prove" binding:"required"`
		APIKey string `form:"api_key" binding:"required"`
	}
	err := c.Bind(&req)
	if err != nil {
		webError(c, -10, err.Error())
		return
	}

	tokens := strings.Split(req.Prove, ".")
	if len(tokens) != 3 {
		webError(c, -20, "Invalid parameter prove")
		return
	}

	xRaw := tokens[0]
	yRaw := tokens[1]
	hRaw := tokens[2]
	//nRaw := tokens[3]

	x, xSuccess := new(big.Int).SetString(xRaw, 16)
	if xSuccess != true {
		webError(c, -24, "Invalid parameter x")
		return
	}

	site, err := siteGet(req.APIKey)
	if err != nil {
		webError(c, -25, "No such site. Invalid api_key? "+err.Error())
		return
	}

	// 2. 验证签名是否正确，同时检查 nonce 是否已使用
	ourH := hmac.New(sha256.New, site.HMACKey).Sum([]byte(x.Text(16)))
	ourHB64 := base64.RawURLEncoding.EncodeToString(ourH)
	if ourHB64 != hRaw {
		webError(c, -30, "Invalid signature for x")
		return
	}

	// 2.2 检查 nonce 是否已经使用
	if nonceIsExists(hRaw) {
		webError(c, -40, "This proof is already used")
		return
	}

	// 3. 检查计算结果是否正确
	isCorrect := false
	var v *vdf.VDF
	ts := time.Now().Unix()
	x, successX := new(big.Int).SetString(xRaw, 16)
	y, successY := new(big.Int).SetString(yRaw, 16)
	if successX != true || successY != true {
		webError(c, -30, "Invalid x or y")
		return
	}

	/// 3.1 使用最新的 RSAKey 检查结果是否正确
	stime := time.Now()

	if ts-site.RSAKeyCreateTime < RSA_KEY_TTL {
		v = vdf.New(site.RSAKey.Primes[0], site.RSAKey.Primes[1])

		if v.Verify(x, site.Hardness, y) == true {
			// 验证成功，什么都不用做
			isCorrect = true
		} else {
			isCorrect = false
		}

		log.Printf("校验证明耗时 %v，模数长度为 %d", verifyElapsed, site.RSAKey.Primes[0].BitLen()*2)
	} else {
		log.Printf("网站 %s 的 RSAKey 已经超时了，不会使用 RSAKey 进行检查", site.APIKey)
	}

	/// 3.2 使用次新的 RSAKey 检查结果是否正确
	if isCorrect == false {
		if ts-site.OldRSAKeyCreateTime > RSA_KEY_TTL*2 {
			v = vdf.New(site.OldRSAKey.Primes[0], site.OldRSAKey.Primes[1])
			if v.Verify(x, site.Hardness, y) != true {
				isCorrect = false
			} else {
				isCorrect = true
			}
		} else {
			log.Printf("网站 %s 的 OldRSAKey 已经超时了，不会使用 OldRSAKey 进行检查", site.APIKey)
		}
	}

	verifyElapsed = time.Now().Sub(stime)

	// 5. 若验证成功则记录一次 nonce
	var msg string
	if isCorrect {
		nonceSet(hRaw)
		msg = fmt.Sprintf("Prove is correct. Verification takes %v", verifyElapsed)
	} else {
		msg = "Prove is INVALID"
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": msg,
		"data": gin.H{
			"prove":          req.Prove,
			"is_correct":     isCorrect,
			"verify_time_ms": float64(verifyElapsed.Microseconds()) / 1000,
		},
	})
}

// 返回一个错误 JSON
func webError(c *gin.Context, code int, msg string) {
	c.JSON(200, gin.H{
		"code":    code,
		"message": msg,
	})
}

func Hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}