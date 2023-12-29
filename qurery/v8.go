package qurery

import (
	"io"
	"os"
	"strings"

	esv8 "github.com/elastic/go-elasticsearch/v8"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

type V8 struct {
	cli *esv8.Client
}

func (v *V8) SearchByFilePath(index string, filePaths ...string) (map[string][]Document, error) {
	var hashs []string
	for _, p := range filePaths {
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

	return v.SearchByHash(index, hashs...)
}

func (v *V8) SearchByHash(index string, hashes ...string) (map[string][]Document, error) {
	var docss [][]Document
	queryStr := generateQueryStr(index, hashes...)

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
	if err := parseResp(data, func(i int, hits map[string]interface{}) error {
		var docs []Document
		for _, hit := range hits["hits"].([]interface{}) {
			var doc Document
			hitMap := hit.(map[string]interface{})
			source := hitMap["_source"].(map[string]interface{})
			srcBytes, err := jsoniter.Marshal(source)
			if err != nil {
				return err
			}
			err = jsoniter.Unmarshal(srcBytes, &doc)
			if err != nil {
				return err
			}
			docs = append(docs, doc)
		}
		docss = append(docss, docs)
		return nil
	}); err != nil {
		return nil, err
	}

	m := map[string][]Document{}
	for i, h := range hashes {
		m[h] = docss[i]
	}

	return m, err
}

// SearchPkgVuln 返回pkg对应的漏洞编号列表
func (v *V8) SearchPkgVuln(index string, pkms ...*PkgKeyMessage) (map[*PkgKeyMessage][]string, error) {
	if err := v.getCpe(index, pkms...); err != nil {
		return nil, err
	}

	queryStr, err := GenerateQueryForVuln(index, pkms...)
	if err != nil {
		return nil, err
	}

	res, err := v.cli.Msearch(strings.NewReader(queryStr))
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return v.checkVuln(data, pkms)
}

func (v *V8) checkVuln(data []byte, pkms []*PkgKeyMessage) (map[*PkgKeyMessage][]string, error) {
	var vss [][]*VulnComponent
	var result = make(map[*PkgKeyMessage][]string, len(pkms))

	if err := parseResp(data, func(i int, hits map[string]interface{}) error {
		var vs []*VulnComponent
		for _, hit := range hits["hits"].([]interface{}) {
			v := new(VulnComponent)
			hitMap := hit.(map[string]interface{})
			source := hitMap["_source"].(map[string]interface{})
			srcBytes, err := jsoniter.Marshal(source)
			if err != nil {
				return err
			}
			if err := jsoniter.Unmarshal(srcBytes, v); err != nil {
				continue
			}
			vs = append(vs, v)
		}
		vss = append(vss, vs)
		return nil
	}); err != nil {
		return nil, err
	}

	if len(vss) != len(pkms) {
		return nil, errors.New("the search does not correspond to the number of responses")
	}

	for i, pkm := range pkms {
		if pkm == nil {
			continue
		}
		for _, v := range vss[i] {
			if v == nil {
				continue
			}
			if IsInfluenced(pkm.Version, v.VulVersionRange) {
				xmirrorIds := result[pkm]
				xmirrorIds = append(xmirrorIds, v.XmirrorId)
				result[pkm] = xmirrorIds
			}
		}
	}

	return result, nil
}

func (v *V8) getCpe(index string, pkms ...*PkgKeyMessage) error {
	q, ids, err := generateCpeQuery(index, pkms...)
	if err != nil {
		return err
	}
	if q == "" {
		return nil
	}

	res, err := v.cli.Msearch(strings.NewReader(q))
	if err != nil {
		return err
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := parseResp(data, func(i int, hits map[string]interface{}) error {
		pkmIndex := ids[i]
		if pkms[pkmIndex] == nil {
			return nil
		}
		var cpes []string
		for _, hit := range hits["hits"].([]interface{}) {
			var c cpe
			hitMap := hit.(map[string]interface{})
			source := hitMap["_source"].(map[string]interface{})
			srcBytes, err := jsoniter.Marshal(source)
			if err != nil {
				return err
			}
			if err := jsoniter.Unmarshal(srcBytes, &c); err != nil {
				continue
			}
			if c.Id != "" {
				cpes = append(cpes, c.Id)
			}
		}
		pkms[pkmIndex].cpes = cpes
		return nil
	}); err != nil {
		return err
	}

	return nil
}
