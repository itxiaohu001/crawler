package flags

import (
	"flag"
	"os"
)

var (
	delay           int
	tool, file, url string
)

type Params struct {
	Delay           int
	Tool, File, Url string
}

func LoadParams() *Params {
	flag.StringVar(&tool, "tool", "", `使用下载工具 例如"wget"`)
	flag.StringVar(&file, "file", "history.log", `下载历史文件路径(如果有的话)`)
	flag.StringVar(&url, "url", "", `下载网址`)
	flag.IntVar(&delay, "delay", 10, `延迟时间 "单位：100*millisecond"`)
	flag.Parse()

	if tool == "" || url == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}

	return &Params{
		Tool:  tool,
		Delay: delay,
		File:  file,
		Url:   url,
	}
}
