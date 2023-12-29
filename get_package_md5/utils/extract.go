package utils

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

func Extract(tp, p string) (string, error) {
	switch tp {
	case "apk":
		parts := strings.Split(p, "/")
		for i, part := range parts {
			if part == "alpine" {
				os := strings.TrimSpace("alpine " + strings.TrimPrefix(parts[i+1], "v"))
				return os, nil
			}
		}
	case "dpkg":
		base := filepath.Base(p)
		return extractUbuntuVersion(base)
	case "rpm":
		parts := strings.Split(p, "/")
		for i, part := range parts {
			if part == "centos" {
				os := strings.TrimSpace("centos " + parts[i+1])
				return os, nil
			}
		}
	}

	return "", errors.New("none os")
}

func extractUbuntuVersion(packageVersion string) (string, error) {
	ok := false
	for _, v := range ubuntuVers {
		if strings.Contains(packageVersion, v) {
			ok = true
			break
		}
	}

	if !ok {
		return "", nil
	}

	re := regexp.MustCompile(`ubuntu(\d+\.\d+(\.\d+)?)[-_]`)

	// 在版本号字符串中搜索匹配项
	matches := re.FindStringSubmatch(packageVersion)

	if len(matches) >= 2 {
		// 返回第一个捕获组（完整的Ubuntu版本号）
		return matches[1], nil
	}

	// 如果没有找到匹配项，返回错误
	return "", fmt.Errorf("Ubuntu version not found in package version: %s", packageVersion)
}

var ubuntuVers = []string{
	"4.10",
	"5.04",
	"5.10",
	"6.06",
	"6.10",
	"7.04",
	"7.10",
	"8.04",
	"8.10",
	"9.04",
	"9.10",
	"10.04",
	"10.10",
	"11.04",
	"11.10",
	"12.04",
	"12.04-ESM",
	"12.10",
	"13.04",
	"13.10",
	"14.04",
	"14.04-ESM",
	"14.10",
	"15.04",
	"15.10",
	"16.04",
	"16.04-ESM",
	"16.10",
	"17.04",
	"17.10",
	"18.04",
	"18.10",
	"19.04",
	"19.10",
	"20.04",
	"20.10",
	"21.04",
	"21.10",
	"22.04",
	"22.10",
}
