package main

import (
	"crypto/tls"
	"get_package_md5/apk_read/byhttp/collector"
	"get_package_md5/apk_read/byhttp/flags"
	"log"
	"net/http"
	"time"
)

func main() {
	p := flags.LoadParams()

	c := collector.NewCollector(DefaultHttpCli(), p.Tool, p.Delay)
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
