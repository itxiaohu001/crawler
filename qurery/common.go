package qurery

import (
	"bytes"
	"crypto/md5"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"io"
)

func generateQueryStr(hs []string, index string) string {
	var buf bytes.Buffer

	for _, h := range hs {
		meta := []byte(fmt.Sprintf(`{ "index" : "%s" }%s`, index, "\n"))
		query := []byte(fmt.Sprintf(`{ "query" : { "term": { "hashes.key": "%s" } } }%s`, h, "\n"))
		buf.Grow(len(meta) + len(query))
		buf.Write(meta)
		buf.Write(query)
	}

	return buf.String()
}

func parseRespData(data []byte) ([][]Document, error) {
	var documents [][]Document
	var r map[string]interface{}

	if err := jsoniter.Unmarshal(data, &r); err != nil {
		return nil, errors.WithMessagef(err, "unmarshal response data")
	}
	resp, ok := r["responses"]
	if !ok {
		return nil, errors.New(`tag "response" not found in response`)
	}
	items, ok := resp.([]interface{})
	if !ok {
		return nil, errors.New(`items is not "[]interface"`)
	}
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, errors.New(`item is not "map[string]interface{}"`)
		}
		hitsMap, ok := itemMap["hits"]
		if !ok {
			return nil, errors.New(`tag "hits" not found in hitsMap`)
		}
		hits, ok := hitsMap.(map[string]interface{})
		if !ok {
			return nil, errors.New(`hitsMap is not "map[string]interface{}"`)
		}
		// 省略了部分断言
		var docs []Document
		for _, hit := range hits["hits"].([]interface{}) {
			hitMap := hit.(map[string]interface{})
			source := hitMap["_source"].(map[string]interface{})
			var doc Document
			srcBytes, err := jsoniter.Marshal(source)
			if err != nil {
				return nil, err
			}
			err = jsoniter.Unmarshal(srcBytes, &doc)
			if err != nil {
				return nil, err
			}
			docs = append(docs, doc)
		}
		documents = append(documents, docs)
	}

	return documents, nil
}

func calculate(r io.Reader) (string, error) {
	h := md5.New()
	_, err := io.Copy(h, r)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
