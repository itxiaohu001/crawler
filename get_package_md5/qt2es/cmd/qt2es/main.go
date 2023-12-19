package main

import (
	"bufio"
	"fmt"
	"get_package_md5/qt2es/inserter"
	"get_package_md5/qt2es/model"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	args := os.Args
	if len(args) < 3 {
		fmt.Printf("Usage: qt2es {filepath} {es_host}")
	}

	f, err := os.Open(args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	errFile, err := os.OpenFile("err.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer errFile.Close()

	escli, err := inserter.NewEsCli(args[2])
	if err != nil {
		log.Fatal(err)
	}

	s := NewSender(escli)
	qtMap := map[string]model.Document{}

	sc := bufio.NewScanner(f)
	cout := 0
	for sc.Scan() {
		if sc.Err() == io.EOF {
			break
		}
		if sc.Err() != nil {
			errFile.WriteString(sc.Err().Error())
		}

		parts := strings.Split(sc.Text(), "|||")
		if len(parts) < 8 {
			errFile.WriteString(sc.Text() + "\n")
			continue
		}
		key := parts[0] + parts[1] + parts[2] + parts[3]
		if v, ok := qtMap[key]; ok {
			v.Hashes = append(v.Hashes, model.Hash{Key: parts[6], Value: parts[5]})
			v.License = append(v.License, parts[4])
			qtMap[key] = v
		} else {
			qtMap[key] = model.Document{
				Manager:      "qt",
				Homepage:     "https://www.qt.io",
				Maintainer:   "qt",
				Name:         parts[0],
				Version:      parts[1],
				Os:           parts[2],
				Architecture: parts[3],
				License:      []string{parts[4]},
				Hashes:       []model.Hash{{Key: parts[6], Value: parts[5]}},
				Description:  parts[7],
			}
		}
		cout++
	}
	log.Printf("一共%d条hash\n", cout)

	log.Printf("开始入库\n")
	cout = 0
	for _, v := range qtMap {
		v.License = uniqueStrings(v.License)
		s.Add(v)
		if s.Ok() {
			if err := s.Send(); err != nil {
				errFile.WriteString(err.Error() + "\n")
			}
			// 防止插入速度过快，给es带来负担
			time.Sleep(time.Second * 1)
			s.Reset()
		}
		cout++
	}

	if s.notEmpty() {
		if err := s.Send(); err != nil {
			errFile.WriteString(err.Error() + "\n")
		}
	}
	log.Printf("入库完毕，一共%d条记录\n", cout)
}

func uniqueStrings(strings []string) []string {
	unique := make(map[string]bool)
	var result []string

	for _, str := range strings {
		if _, ok := unique[str]; !ok {
			unique[str] = true
			result = append(result, str)
		}
	}

	return result
}

// Max 一次插入五百条数据
const Max = 500

type Sender struct {
	es      *inserter.EsCli
	qtInfos []model.Document
	l       int
}

func NewSender(cli *inserter.EsCli) *Sender {
	return &Sender{
		es:      cli,
		qtInfos: []model.Document{},
		l:       0,
	}
}

func (s *Sender) Ok() bool {
	return s.l >= Max
}

func (s *Sender) Reset() {
	s.qtInfos = nil
	s.l = 0
}

func (s *Sender) Add(info model.Document) {
	s.qtInfos = append(s.qtInfos, info)
	s.l++
}

func (s *Sender) Send() error {
	return s.es.Insert(s.qtInfos)
}

func (s *Sender) notEmpty() bool {
	return s.l > 0
}
