package flags

import (
	"flag"
	"os"
)

var (
	delay     int
	file, url string
)

type Params struct {
	Delay     int
	File, Url string
}

func LoadParams() *Params {
	// todo:支持正则过滤
	flag.StringVar(&url, "url", "", `下载网址`)
	flag.IntVar(&delay, "delay", 10, `延迟时间 "单位：100*millisecond"`)
	flag.Parse()

	if url == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}

	return &Params{
		Delay: delay,
		File:  file,
		Url:   url,
	}
}
