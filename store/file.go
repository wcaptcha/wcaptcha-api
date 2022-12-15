package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type File struct {
	dir string
}

// Init Storage
func (f *File) Init() error {
	var err error

	f.dir = os.Getenv("STORAGE_PATH")
	if len(f.dir) == 0 {
		return errors.New("必须指定 STORAGE_PATH 环境变量")
	}

	if f.dir[len(f.dir)-1:] != "/" {
		f.dir += "/"
	}

	err = os.MkdirAll(f.dir, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// List objects by prefix
func (f *File) List(prefix string) ([]string, error) {
	p := prefix
	if len(p) > 0 && p[0:1] == "/" {
		p = p[1:]
	}
	if len(p) > 0 && p[len(p)-1:] == "/" {
		p = p[0 : len(p)-1]
	}

	entries, err := os.ReadDir(f.dir + p)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		} else {
			return nil, err
		}
	}

	var ret []string
	for _, v := range entries {
		if v.IsDir() {
			continue
		}

		ret = append(ret, p+"/"+v.Name())
	}

	return ret, nil
}

// Put an object into storage
func (f *File) Put(key string, obj any) error {
	err := f.ensureFileDir(key)
	if err != nil {
		return fmt.Errorf("Put() failed: %v", err)
	}

	fi, err := os.OpenFile(f.dir+key, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer fi.Close()

	e := json.NewEncoder(fi)
	e.SetIndent("", "    ")
	err = e.Encode(obj)

	if err != nil {
		return err
	}

	return err
}

// Get an object from storage
func (f *File) Get(key string, obj any) error {
	fi, err := os.OpenFile(f.dir+key, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer fi.Close()

	d := json.NewDecoder(fi)

	err = d.Decode(&obj)

	return err
}

// Check if an object is in storage
func (f *File) KeyExists(key string) (bool, error) {
	_, err := os.Stat(f.dir + key)
	if err == nil {
		return true, nil
	} else {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}
}

func (f *File) Delete(key string) error {
	return os.Remove(f.dir + key)
}

func (f *File) ensureFileDir(key string) error {
	p := f.dir + key
	tokens := strings.Split(p, "/")
	if len(tokens) > 0 {
		tokens = tokens[0 : len(tokens)-1]
	}
	p = strings.Join(tokens, "/")

	err := os.MkdirAll(p, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Unable to ensureFileDir `%s': %v", p, err)
	} else {
		return nil
	}
}
