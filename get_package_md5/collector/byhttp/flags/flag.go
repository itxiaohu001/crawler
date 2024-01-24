package flags

import (
	"flag"
	"strings"
)

var (
	TypeList []string
	Limit    int
	Out      string
	Cache    string
)

func LoadFlags() {
	var list string
	flag.StringVar(&list, "pkg", "", "指定爬取类型(alpine、centos、ubuntu、debian)，用逗号分开，默认全部爬取")
	flag.IntVar(&Limit, "l", 8, "协程数限制")
	flag.StringVar(&Out, "o", "./", "结果保存位置")
	flag.StringVar(&Cache, "c", "./cache", "下载缓存目录")

	flag.Parse()

	if list == "" {
		TypeList = []string{"alpine", "centos", "ubuntu", "debian"}
	} else {
		if !strings.Contains(list, ",") {
			TypeList = []string{list}
		} else {
			for _, pkgType := range strings.Split(list, ",") {
				TypeList = append(TypeList, pkgType)
			}
		}
	}
}
