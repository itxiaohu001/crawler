package flags

import (
	"flag"
	"os"
)

var (
	delay                   int
	suffix, tool, file, url string
)

type Params struct {
	Delay                   int
	Suffix, Tool, File, Url string
}

func LoadParams() *Params {
	// todo:支持正则过滤
	flag.StringVar(&suffix, "suffix", "", `需要下载的文件后缀，例如(.apk .rpm .deb .zst)`)
	flag.StringVar(&tool, "tool", "", `使用下载工具 例如"wget"`)
	flag.StringVar(&url, "url", "", `下载网址`)
	flag.IntVar(&delay, "delay", 10, `延迟时间 "单位：100*millisecond"`)
	flag.Parse()

	if tool == "" || url == "" || suffix == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}

	return &Params{
		Tool:   tool,
		Delay:  delay,
		File:   file,
		Url:    url,
		Suffix: suffix,
	}
}
