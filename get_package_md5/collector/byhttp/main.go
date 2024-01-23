package main

import (
	"crypto/tls"
	"get_package_md5/collector/byhttp/collector"
	"get_package_md5/collector/byhttp/recorder"
	sleeper2 "get_package_md5/collector/byhttp/sleeper"
	"get_package_md5/parser"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	pool2 "github.com/sourcegraph/conc/pool"
)

const (
	_historyFilePath = "download_history.log"
	_errorFilePath   = "error.log"
	_limit           = 8
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage %s: %s [remote address...]", filepath.Base(os.Args[0]), filepath.Base(os.Args[0]))
	}

	// 注册解析器
	parsers := []parser.Parser{parser.NewDebParser(), parser.NewApkParser(), parser.NewRpmParser()}

	// 创建recorder
	rcd, err := recorder.NewAccessRecorder(_historyFilePath, _errorFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer rcd.Close()

	// 创建goroutine池
	pool := pool2.New().WithMaxGoroutines(_limit)

	// 创建sleeper防止访问过快
	sleeper := sleeper2.NewSleeper(500*time.Microsecond, 500*time.Microsecond)

	// 创建collector
	c := collector.NewCollector(DefaultHttpCli(), parsers, rcd, pool, sleeper)

	for _, u := range os.Args[1:] {
		if err := c.Visit(u, func(packageUrl string) {
			if err := c.DownloadAndParse(packageUrl); err != nil {
				c.Recorder.RecordError(err.Error())
			}
		}); err != nil {
			log.Println(err)
		}
	}
	c.Wait()
}

func DefaultHttpCli() http.Client {
	return http.Client{
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			IdleConnTimeout:   30 * time.Second, // 空闲连接的超时时间
			DisableKeepAlives: false,            // 长连接
		},
	}
}
