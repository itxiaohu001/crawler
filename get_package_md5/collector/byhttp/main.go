package main

import (
	"crypto/tls"
	"get_package_md5/collector/byhttp/collector"
	"get_package_md5/parser"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage %s: %s {remote address}", filepath.Base(os.Args[0]), filepath.Base(os.Args[0]))
	}
	parsers := []parser.Parser{parser.NewDebParser(), parser.NewApkParser(), parser.NewRpmParser()}
	c := collector.NewCollector(DefaultHttpCli(), parsers, 10)
	if err := c.Visit(os.Args[1]); err != nil {
		log.Fatal(err)
	}
}

func demo() {
	//prefix := "https://mirrors.ustc.edu.cn/alpine/"
	//list := []string{"v3.7", "v3.8", "v3.9", "v3.10", "v3.11",
	//	"v3.12", "v3.13", "v3.14", "v3.15", "v3.16", "v3.17",
	//	"v3.18", "v3.19"}
	//parsers := []parser.Parser{parser.NewApkParser()}

	prefix := "https://mirrors.ustc.edu.cn/ubuntu/pool/universe/"
	list := []string{"k", "l", "lib2", "lib3", "liba", "libb", "libc",
		"libd", "libe", "libf", "libg", "libh", "libi", "libj", "libk",
		"libl", "libm", "libn", "libo", "libp", "libq", "libr", "libs",
		"libt", "libu", "libv", "libw", "libx", "liby", "libz", "m", "n",
		"o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	parsers := []parser.Parser{parser.NewDebParser()}
	c := collector.NewCollector(DefaultHttpCli(), parsers, 10)
	for _, suffix := range list {
		url := prefix + suffix
		if err := c.Visit(url); err != nil {
			log.Fatal(err)
		}
	}
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
