package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"github.com/gocolly/colly"
	"github.com/pkg/errors"
	"goLearning/get_package_md5/utils"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

const (
	_trunc  = os.O_TRUNC | os.O_CREATE | os.O_RDWR
	_append = os.O_APPEND | os.O_CREATE | os.O_RDWR
)

var (
	urls             []string
	errLog           = "apk_error_visit.log"
	skipSuffix       = []string{"..", ".iso", "releases", "APKINDEX.tar.gz"}
	successSavedFile = "apk_success_downloaded"
	successMap       = make(map[string]struct{}, 100000)
)

func init() {
	if _, err := os.Stat(successSavedFile); err != nil {
		if err != os.ErrNotExist {
			log.Fatal(err)
		}
		if f, err := os.OpenFile(successSavedFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644); err != nil {
			log.Fatal(err)
		} else {
			f.Close()
		}
	} else {
		f, _ := os.Open(successSavedFile)
		sc := bufio.NewScanner(f)
		log.Printf("loading %s", successSavedFile)
		for sc.Scan() {
			successMap[strings.TrimSpace(sc.Text())] = struct{}{}
		}
		f.Close()
		log.Printf("loading %s success,%d records in all", successSavedFile, len(successMap))
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: apk_repo_collector urls...")
		return
	}
	for _, url := range os.Args[1:] {
		urls = append(urls, url)
	}
	log.Printf("start to visit %+v\n", urls)

	c := initCollector()
	c.OnRequest(func(request *colly.Request) {
		log.Printf("visit: %s\n", request.URL.String())
	})
	c.OnResponse(func(response *colly.Response) {
		saveName := "." + response.Request.URL.Path
		if !strings.HasSuffix(response.FileName(), ".apk") {
			return
		}
		if err := CreateAndSave(saveName, response.Body, _trunc); err != nil {
			utils.RecordErrors(err, errLog)
			return
		}
		utils.RecordDownloaded(response.Request.URL.String(), successSavedFile)
	})
	c.OnError(func(response *colly.Response, err error) {
		if response.StatusCode == 403 || response.StatusCode == 426 {
			utils.RecordErrors(errors.Wrapf(err, "visit %s was banned", response.Request.URL.String()), errLog)
			os.Exit(0)
		} else {
			utils.RecordErrors(errors.Wrapf(err, "visit %s error", response.Request.URL.String()), errLog)
		}
	})
	c.OnHTML("pre a", func(element *colly.HTMLElement) {
		name := strings.Trim(element.Attr("href"), "/")
		for _, v := range skipSuffix {
			if strings.HasSuffix(name, v) {
				return
			}
		}
		addr := element.Request.URL.String() + name
		if _, ok := successMap[strings.TrimSpace(addr)]; ok {
			return
		}
		try(c, 3, addr)
	})

	for _, url := range urls {
		c.Visit(url)
	}
}

// try 进行1至times次访问
func try(c *colly.Collector, times int, addr string) {
	for i := 0; i < times; i++ {
		err := c.Visit(addr)
		if err == nil {
			return
		}
	}
}

func initCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.AllowedDomains("dl-cdn.alpinelinux.org"),
		colly.UserAgent("Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:86.0) Gecko/20100101 Firefox/86.0"),
		colly.AllowURLRevisit(),
		colly.MaxDepth(4),
	)
	c.WithTransport(&http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		IdleConnTimeout:   30 * time.Second, // 空闲连接的超时时间
		DisableKeepAlives: false,            // 长连接
	})
	c.SetRequestTimeout(time.Second * 30)
	c.Limit(&colly.LimitRule{
		Delay:       2 * time.Second, // 设置延迟时间
		RandomDelay: 2 * time.Second, // 添加随机延迟
		Parallelism: 1,
	})
	return c
}

func CreateAndSave(locate string, data []byte, flag int) error {
	dir, _ := path.Split(locate)
	if err := os.MkdirAll(dir, 0644); err != nil {
		return errors.Wrapf(err, "create dir: %s", dir)
	}
	f, err := os.OpenFile(locate, flag, 0644)
	if err != nil {
		return errors.Wrapf(err, "open file: %s", locate)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return errors.Wrapf(err, "write data to: %s", locate)
	}
	return nil
}
