package store

import (
	"fmt"
	"testing"
)

func TestFile(t *testing.T) {
	testItem(t, "abc")
	testItem(t, "a/b/c")
	testItem(t, "/a/b/c")
}

func TestList(t *testing.T) {
	var err error
	Store := new(File)
	if err != nil {
		t.FailNow()
	}

	Store.Put("d1/foo", "bar")

	l, err := Store.List("d1")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	if len(l) != 1 || l[0] != "d1/foo" {
		fmt.Println(l)
		t.FailNow()
	}

	l, err = Store.List("d1/")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	if len(l) != 1 || l[0] != "d1/foo" {
		t.FailNow()
	}

	l, err = Store.List("d1_not_exists/")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	if len(l) != 0 {
		t.FailNow()
	}

}

func testItem(t *testing.T, key string) {
	var err error

	Store := new(File)
	err = Store.Init()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	err = Store.Put(key, []string{"foo", "bar"})
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	var obj []string
	err = Store.Get(key, &obj)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	if len(obj) != 2 {
		t.FailNow()
	}
	if obj[0] != "foo" || obj[1] != "bar" {
		t.FailNow()
	}

	b, err := Store.KeyExists(key)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	if b != true {
		t.FailNow()
	}

	b, err = Store.KeyExists("not_exists")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	if b != false {
		t.FailNow()
	}

	err = Store.Delete(key)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	b, err = Store.KeyExists(key)
	if b == true && err != nil {
		fmt.Println(err)
		t.FailNow()
	}
}
