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
	var dir = "E:\\tmp\\linux_pkg_hash\\deb\\ubuntu"
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

		var m model.DebPkg
		o, _ := utils.Extract("dpkg", path)
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
	slice []model.DebPkg
	bar   *utils.Bar
	es    *es.Cli
}

func NewSender(b *utils.Bar, cli *es.Cli) *sender {
	return &sender{
		mu:    sync.Mutex{},
		cur:   0,
		slice: []model.DebPkg{},
		bar:   b,
		es:    cli,
	}
}

func (l *sender) ready() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.cur == 1000
}

func (l *sender) add(a model.DebPkg) {
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
	l.slice = []model.DebPkg{}
}

func (l *sender) send() {
	defer func() {
		l.clear()
	}()

	var docs []model.Document
	for _, v := range l.slice {
		var doc model.Document
		if len(v.Hashes) < 1 {
			continue
		}

		doc.Os = v.OS
		doc.Manager = "dpkg"
		doc.Name = v.Name
		doc.Version = v.Version
		doc.Depends = v.Depends
		doc.Architecture = v.Architecture
		doc.Description = v.Description
		for k, v := range v.Hashes {
			doc.Hashes = append(doc.Hashes, model.Hash{Key: k, Value: v})
		}
		for _, l := range v.Licences {
			doc.License = append(doc.License, l.Names...)
		}
		doc.Maintainer = v.Maintainer
		doc.Homepage = v.Homepage
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
