package main

import (
	"get_package_md5/es"
	"get_package_md5/model"
	"get_package_md5/utils"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

func main() {
	var dir = "E:\\tmp\\linux_pkg_hash\\apk\\alpine"
	var bar = utils.NewBar(total(dir))
	esCli, err := es.NewEsCli("http://127.0.0.1:9200")
	if err != nil {
		log.Fatal(err)
	}
	var l = NewSender(bar, esCli)

	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return errors.WithMessagef(err, "open %s", path)
		}
		defer f.Close()

		var m model.ApkPkg
		o, _ := utils.Extract("apk", path)
		if o != "" {
			m.OS = o
		}
		if err := jsoniter.NewDecoder(f).Decode(&m); err != nil {
			return errors.WithMessagef(err, "decode %s", path)
		}

		l.add(m)
		if l.ready() {
			l.send()
		}

		return nil
	}); err != nil {
		log.Println("work dir error", err)
	}

	l.send()
}

type sender struct {
	mu    sync.Mutex
	cur   int
	slice []model.ApkPkg
	bar   *utils.Bar
	es    *es.Cli
}

func NewSender(b *utils.Bar, cli *es.Cli) *sender {
	return &sender{
		mu:    sync.Mutex{},
		cur:   0,
		slice: []model.ApkPkg{},
		bar:   b,
		es:    cli,
	}
}

func (l *sender) ready() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.cur == 1000
}

func (l *sender) add(a model.ApkPkg) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.bar.Add()

	l.cur++
	l.slice = append(l.slice, a)
}

func (l *sender) clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cur = 0
	l.slice = []model.ApkPkg{}
}

func (l *sender) send() {
	defer func() {
		l.clear()
	}()

	var docs []model.Document
	for _, v := range l.slice {
		if len(v.Hashes) < 1 {
			continue
		}
		var doc model.Document
		doc.Os = v.OS
		doc.Manager = "apk"
		doc.Name = v.PkgName
		doc.Version = v.PkgVer
		doc.Depends = v.Depend
		doc.Architecture = v.Arch
		doc.Description = v.PkgDesc
		for _, hash := range v.Hashes {
			doc.Hashes = append(doc.Hashes, model.Hash{Key: hash.Key, Value: hash.Value})
		}
		doc.License = v.License
		doc.Maintainer = v.Maintainer
		doc.Homepage = v.URL
		docs = append(docs, doc)
	}

	if err := l.es.Insert(docs); err != nil {
		log.Println(err)
	}

	l.bar.Print()
}

func total(p string) int {
	var t int

	filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".json") {
			t++
		}
		return nil
	})

	return t
}
