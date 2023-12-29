package qurery

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

func generateQueryStr(index string, hashes ...string) string {
	var buf bytes.Buffer
	meta := []byte(fmt.Sprintf(`{ "index" : "%s" }%s`, index, "\n"))

	for _, h := range hashes {
		query := []byte(fmt.Sprintf(`{ "query" : { "term": { "hashes.key": "%s" } } }%s`, h, "\n"))
		buf.Grow(len(meta) + len(query))
		buf.Write(meta)
		buf.Write(query)
	}

	return buf.String()
}

func GenerateQueryForVuln(index string, pkms ...*PkgKeyMessage) (string, error) {
	var buf bytes.Buffer
	var termsMap []map[string]interface{}
	meta := []byte(fmt.Sprintf(`{ "index" : "%s" }%s`, index, "\n"))

	for _, pkm := range pkms {
		if len(pkm.cpes) > 0 {
			termsMap = []map[string]interface{}{
				{"term": map[string]interface{}{"component_name.keyword": pkm.PkgName}},
				{"term": map[string]interface{}{"type": pkm.OS.string()}},
				{"terms": map[string]interface{}{"vul_version_detail": pkm.cpes}},
			}
		} else {
			termsMap = []map[string]interface{}{
				{"term": map[string]interface{}{"component_name.keyword": pkm.PkgName}},
				{"term": map[string]interface{}{"type": pkm.OS.string()}},
			}
		}
		queryMap := map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"filter": termsMap,
				},
			},
			"size": 1000,
		}

		query, err := jsoniter.Marshal(queryMap)
		if err != nil {
			return "", err
		}
		query = append(query, '\n')

		buf.Grow(len(meta) + len(query))
		buf.Write(meta)
		buf.Write(query)
	}

	return buf.String(), nil
}

func generateCpeQuery(index string, pkms ...*PkgKeyMessage) (string, []int, error) {
	var arryIndexs []int
	var buf bytes.Buffer
	meta := []byte(fmt.Sprintf(`{ "index" : "%s" }%s`, index, "\n"))

	for i, pkm := range pkms {
		if len(pkm.repos) > 0 || len(pkm.nvrs) > 0 {
			arryIndexs = append(arryIndexs, i)
		} else {
			continue
		}

		should := make([]map[string]interface{}, 0, 2)
		if len(pkm.repos) > 0 {
			should = append(should, map[string]interface{}{
				"terms": map[string]interface{}{
					"repo": pkm.repos,
				},
			})
		}
		if len(pkm.nvrs) > 0 {
			should = append(should, map[string]interface{}{
				"terms": map[string]interface{}{
					"nvr": pkm.nvrs,
				},
			})
		}

		queryMap := map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"should": should,
				},
			},
			"size": 1000,
		}
		data, err := jsoniter.Marshal(queryMap)
		if err != nil {
			return "", nil, err
		}
		query := append(data, '\n')
		buf.Grow(len(meta) + len(query))
		buf.Write(meta)
		buf.Write(query)
	}

	return buf.String(), nil, nil
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

func parseResp(data []byte, do func(i int, hits map[string]interface{}) error) error {
	var r map[string]interface{}

	if err := jsoniter.Unmarshal(data, &r); err != nil {
		return errors.WithMessagef(err, "unmarshal response data")
	}
	resp, ok := r["responses"]
	if !ok {
		return errors.New(`tag "response" not found in response`)
	}
	items, ok := resp.([]interface{})
	if !ok {
		return errors.New(`items is not "[]interface"`)
	}
	for i, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return errors.New(`item is not "map[string]interface{}"`)
		}
		hitsMap, ok := itemMap["hits"]
		if !ok {
			return errors.New(`tag "hits" not found in hitsMap`)
		}
		hits, ok := hitsMap.(map[string]interface{})
		if !ok {
			return errors.New(`hitsMap is not "map[string]interface{}"`)
		}
		if err := do(i, hits); err != nil {
			return err
		}
	}

	return nil
}

func calculate(r io.Reader) (string, error) {
	h := md5.New()
	_, err := io.Copy(h, r)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

var defaultContentSets = map[string][]string{
	"6": {
		"rhel-6-server-rpms",
		"rhel-6-server-extras-rpms",
	},
	"7": {
		"rhel-7-server-rpms",
		"rhel-7-server-extras-rpms",
	},
	"8": {
		"rhel-8-for-x86_64-baseos-rpms",
		"rhel-8-for-x86_64-appstream-rpms",
	},
	"9": {
		"rhel-9-for-x86_64-baseos-rpms",
		"rhel-9-for-x86_64-appstream-rpms",
	},
}
