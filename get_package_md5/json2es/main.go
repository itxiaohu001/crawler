package main

import (
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"get_package_md5/model"

	es2 "get_package_md5/es"

	jsoniter "github.com/json-iterator/go"
)

var (
	dir   string
	es    string
	index string
	t     string
	idx   int
	// 一次插入一百条数据
	limit     = 100
	limitArry = make([]*model.CommonPkg, limit)
	sum       int64
)

func init() {
	flag.StringVar(&dir, "d", "", "json files directory")
	flag.StringVar(&es, "es", "", "es address")
	flag.StringVar(&index, "index", "", "es索引名")
	flag.StringVar(&t, "type", "", "包管理器类型")
	flag.Parse()

	if t == "" || index == "" || es == "" || dir == "" {
		log.Fatal("Make sure you use the correct parameters")
	}
}

func main() {
	esCli, err := es2.NewClientV2(es)
	if err != nil {
		log.Fatal(err)
	}

	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(d.Name(), ".json") {
			if idx == limit {
				if e := esCli.MultInsert(index, limitArry); e != nil {
					log.Println(e, path)
				}
				sum += int64(limit)
				log.Printf("inserted %d pkgs", sum)
				// 避免数据插入速度太快，es拒绝服务
				time.Sleep(time.Second)
				clear(limitArry)
				idx = 0
			}

			f, e := os.Open(path)
			if e != nil {
				log.Println(err)
			}
			defer f.Close()

			pkg := new(model.DebPkg)
			if e := jsoniter.NewDecoder(f).Decode(pkg); e != nil {
				log.Println(e)
			}
			cp := model.Convert(pkg, t)

			if idx < limit {
				limitArry[idx] = cp
				idx++
				return nil
			}
		}
		return err
	}); err != nil {
		log.Println(err)
	}

	// 处理剩余pkgs
	if limitArry[0] != nil {
		if e := esCli.MultInsert(index, limitArry); e != nil {
			log.Println(e)
		}
	}
}
