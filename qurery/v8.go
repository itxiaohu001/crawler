package qurery

import (
	esv8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/pkg/errors"
	"io"
	"os"
	"strings"
)

type V8 struct {
	cli *esv8.Client
}

func (v *V8) SearchByHash(h, index string) ([]Document, error) {
	queryStr := generateQueryStr([]string{h}, index)

	res, err := v.cli.Msearch(strings.NewReader(queryStr))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		d, _ := io.ReadAll(res.Body)
		return nil, errors.Errorf("%s", d)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	docs, err := parseRespData(data)
	if err != nil {
		return nil, err
	}

	if len(docs) < 1 {
		return nil, nil
	}

	return docs[0], nil
}

func (v *V8) MultiSearch(hs []string, index string) (map[string][]Document, error) {
	queryStr := generateQueryStr(hs, index)
	res, err := v.cli.Msearch(strings.NewReader(queryStr))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		d, _ := io.ReadAll(res.Body)
		return nil, errors.Errorf("%s", d)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	docs, err := parseRespData(data)
	if err != nil {
		return nil, err
	}

	m := map[string][]Document{}
	for i, h := range hs {
		m[h] = docs[i]
	}

	return m, err
}

func (v *V8) SearchByFilePath(p, index string) ([]Document, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h, err := calculate(f)

	return v.SearchByHash(h, index)
}

func (v *V8) MultiSearchByFilePath(ps []string, index string) (map[string][]Document, error) {
	var hashs []string
	for _, p := range ps {
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		h, err := calculate(f)
		if err != nil {
			return nil, err
		}
		hashs = append(hashs, h)
	}

	return v.MultiSearch(hashs, index)
}
