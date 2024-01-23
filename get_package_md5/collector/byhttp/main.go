package main

import (
	"bufio"
	"crypto/tls"
	"get_package_md5/collector/byhttp/collector"
	"get_package_md5/collector/recorder"
	sleeper2 "get_package_md5/collector/sleeper"
	"get_package_md5/parser"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	pool2 "github.com/sourcegraph/conc/pool"
)

const (
	_historyFilePath = "download_history.log"
	_errorFilePath   = "error.log"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage %s: %s [remote address...]", filepath.Base(os.Args[0]), filepath.Base(os.Args[0]))
	}
	var urls []string
	for _, u := range os.Args[1:] {
		uu, err := url.Parse(u)
		if err != nil {
			log.Printf("invalid url %s\n", u)
			continue
		}
		urls = append(urls, uu.String())
	}

	// 注册解析器
	parsers := []parser.Parser{parser.NewDebParser(), parser.NewApkParser(), parser.NewRpmParser()}

	// 加载历史下载记录
	history, err := readHistory(_historyFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// 创建recorder
	rcd, err := recorder.NewAccessRecorder(_historyFilePath, _errorFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer rcd.Close()

	// 注册goroutine池
	pool := pool2.New().WithMaxGoroutines(8)

	// 创建sleeper防止访问过快
	sleeper := sleeper2.NewSleeper(500*time.Microsecond, 500*time.Microsecond)

	// 创建collector
	c := collector.NewCollector(DefaultHttpCli(), parsers, history, rcd, pool, sleeper)

	for _, url := range urls {
		if err := c.Visit(url, func(packageUrl string) {
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

func readHistory(logPath string) (map[string]struct{}, error) {
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	history := map[string]struct{}{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		history[strings.TrimSpace(sc.Text())] = struct{}{}
	}
	log.Printf("%d historical records in all\n", len(history))

	return history, nil
}

func demo() {
	//prefix := "https://mirrors.ustc.edu.cn/alpine/"
	//list := []string{"v3.7", "v3.8", "v3.9", "v3.10", "v3.11",
	//	"v3.12", "v3.13", "v3.14", "v3.15", "v3.16", "v3.17",
	//	"v3.18", "v3.19"}
	//parsers := []parser.Parser{parser.NewApkParser()}
	//
	//prefix := "https://mirrors.ustc.edu.cn/ubuntu/pool/universe/"
	//list := []string{"k", "l", "lib2", "lib3", "liba", "libb", "libc",
	//	"libd", "libe", "libf", "libg", "libh", "libi", "libj", "libk",
	//	"libl", "libm", "libn", "libo", "libp", "libq", "libr", "libs",
	//	"libt", "libu", "libv", "libw", "libx", "liby", "libz", "m", "n",
	//	"o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
}
