package main

import (
	"crypto/tls"
	"get_package_md5/collector/byhttp/collector"
	"get_package_md5/collector/byhttp/flags"
	"get_package_md5/parser"
	"log"
	"net/http"
	"time"
)

func main() {
	p := flags.LoadParams()

	parsers := []parser.Parser{parser.NewDebParser(), parser.NewApkParser(), parser.NewRpmParser()}
	c := collector.NewCollector(DefaultHttpCli(), parsers, p.Delay)
	if err := c.Visit(p.Url); err != nil {
		log.Fatal(err)
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
