package es

import (
	"bytes"
	"fmt"
	"get_package_md5/model"
	"io"
	"net/http"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

type ClientV2 struct {
	url     string
	httpCli http.Client
}

func check(url string) (bool, error) {
	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("connect error %s", url)
	}
	if resp.Body != nil && resp.StatusCode != 200 {
		return false, fmt.Errorf("connect error %s", url)
	}
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	return true, nil
}

func NewClientV2(url string) (*ClientV2, error) {
	url = strings.TrimSuffix(url, "/")
	if ok, err := check(url); !ok {
		return nil, err
	}
	tr := &http.Transport{
		IdleConnTimeout:   30 * time.Second, // 空闲连接的超时时间
		DisableKeepAlives: false,            // 启用长连接
	}
	return &ClientV2{
		strings.TrimSuffix(url, "/"),
		http.Client{Transport: tr},
	}, nil
}

func (c *ClientV2) Insert(index string, pkg *model.CommonPkg) error {
	fullUrl := c.url + "/" + index + "/_doc"
	data, err := jsoniter.Marshal(pkg)
	if err != nil {
		return errors.Wrapf(err, "marshal data")
	}
	req, err := http.NewRequest("POST", fullUrl, bytes.NewReader(data))
	if err != nil {
		return errors.Wrapf(err, "create request for %s", c.url)
	}
	req.Header.Set("Content-Type", "application/json")
	_, err = c.httpCli.Do(req)
	if err != nil {
		return errors.Wrapf(err, "visit %s", c.url)
	}
	return nil
}

func (c *ClientV2) MultInsert(index string, packages []*model.CommonPkg) error {
	var buf bytes.Buffer
	for _, pkg := range packages {
		if pkg == nil {
			break
		}
		if len(pkg.Hashes) == 0 {
			continue
		}
		meta := map[string]map[string]string{"index": {"_index": index}}
		metaJSON, _ := jsoniter.Marshal(meta)
		pkgJSON, _ := jsoniter.Marshal(pkg)
		buf.Write(metaJSON)
		buf.WriteByte('\n')
		buf.Write(pkgJSON)
		buf.WriteByte('\n')
	}
	fullUrl := c.url + "/" + "_bulk"
	req, err := http.NewRequest("POST", fullUrl, &buf)
	if err != nil {
		return errors.Wrapf(err, "create request for %s", c.url)
	}
	req.Header.Set("Content-Type", "application/json")
	_, err = c.httpCli.Do(req)
	if err != nil {
		return errors.Wrapf(err, "visit %s", c.url)
	}
	return nil
}

func (c *ClientV2) Search(index string, key string) (*model.DebPkg, error) {
	fullUrl := c.url + "/" + index + "/_search"
	queryStr := fmt.Sprintf(`
	{
		"query": {
    		"term": {
      			"hashes.key": %s
    		}
		}
	}`, key)
	req, err := http.NewRequest("POST", fullUrl, bytes.NewReader([]byte(queryStr)))
	if err != nil {
		return nil, errors.Wrapf(err, "create request for %s", c.url)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "visit %s", c.url)
	}
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "read body")
	}
	fmt.Println(string(data))

	return nil, nil
}
