package store

import (
	"os"

	"github.com/greensea/s3kv"
)

type S3 struct {
	s3 *s3kv.Storage
}

// Init Storage
func (s *S3) Init() error {
	var err error
	s.s3, err = s3kv.New(&s3kv.Config{
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		Bucket:    os.Getenv("S3_BUCKET"),
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
	})
	return err
}

// List objects by prefix
func (s *S3) List(prefix string) ([]string, error) {
	ret, err := s.s3.List(prefix)
	return ret, err
}

// Put an object into storage
func (s *S3) Put(key string, obj any) error {
	return s.s3.PutObject(key, obj)
}

// Get an object from storage
func (s *S3) Get(key string, obj any) error {
	return s.s3.GetJSON(key, obj)
}

// Check if an object is in storage
func (s *S3) KeyExists(key string) (bool, error) {
	return s.s3.KeyExists(key)
}

func (s *S3) Delete(key string) error {
	return s.s3.Delete(key)
}
