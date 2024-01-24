package main

import (
	"crypto/tls"
	"get_package_md5/collector/byhttp/collector"
	"get_package_md5/collector/byhttp/flags"
	"get_package_md5/collector/byhttp/recorder"
	sleeper2 "get_package_md5/collector/byhttp/sleeper"
	"log"
	"net/http"
	"time"

	pool2 "github.com/sourcegraph/conc/pool"
)

func main() {
	flags.LoadFlags()
	collector.Register(flags.TypeList)

	// 创建recorder
	rcd, err := recorder.NewAccessRecorder(flags.Cache)
	if err != nil {
		log.Fatal(err)
	}
	defer rcd.Close()

	// 创建goroutine池
	pool := pool2.New().WithMaxGoroutines(flags.Limit)

	// 创建sleeper防止访问过快
	sleeper := sleeper2.NewSleeper(500*time.Microsecond, 500*time.Microsecond)

	// 创建collector
	c := collector.NewCollector(DefaultHttpCli(), rcd, pool, sleeper, flags.Out)

	c.Start(func(packageUrl string) {
		err := c.DownloadAndParse(packageUrl)
		if err != nil {
			c.Recorder.RecordError(err.Error())
		}
	})
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
