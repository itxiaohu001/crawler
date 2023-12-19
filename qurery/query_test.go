package qurery

import (
	"fmt"
	"testing"
)

func TestSearch(t *testing.T) {
	address := "http://172.25.0.22:8017"
	searcher, err := NewSearcher(WithConfig(&Config{
		Version: Es7,
		Address: []string{address},
	}))
	if err != nil {
		t.Error(err)
		return
	}

	res, err := searcher.SearchByHash("ee92762b60b98857bf1a00d1f9666b58", "pkg_bin_hash_final")
	if err != nil {
		t.Error(err)
		return
	}
	for _, doc := range res {
		fmt.Printf("%+v\n", doc)
	}

	multiRes, err := searcher.MultiSearch([]string{"ee92762b60b98857bf1a00d1f9666b58", "cb8481f37763b2ae98b1217e793e8f3f"}, "pkg_bin_hash_final")
	if err != nil {
		t.Error(err)
		return
	}
	for key, doc := range multiRes {
		fmt.Printf("key %s, val %+v\n", key, doc)
	}
}
