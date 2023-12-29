package qurery

import (
	"strings"

	version2 "github.com/knqyf263/go-deb-version"
)

func IsInfluenced(ver string, ranges string) bool {
	if ranges == "" {
		// 无fix则认定受漏洞影响
		return true
	}
	for _, expr := range strings.Split(ranges, "||") {
		expr = strings.TrimSpace(expr)
		if InRange(ver, expr) {
			return true
		}
	}
	return false
}

func InRange(ver string, expr string) bool {
	if len(ver) == 0 || len(expr) == 0 {
		return false
	}
	trimmedVer := strings.TrimPrefix(ver, "v")
	version, err := version2.NewVersion(trimmedVer)
	if err != nil {
		return false
	}

	lr := strings.Split(strings.Trim(expr, "[]()"), ",")
	if len(lr) != 2 {
		return false
	}
	if lr[0] == "" {
		lr[0] = "0"
	}

	leftVersion := strings.TrimPrefix(lr[0], "v")
	rightVersion := strings.TrimPrefix(lr[1], "v")
	left, errLeft := version2.NewVersion(leftVersion)
	right, errRight := version2.NewVersion(rightVersion)
	if errLeft != nil || errRight != nil {
		return false
	}

	start, end := expr[0], expr[len(expr)-1]
	switch {
	case start == '[' && end == ']':
		return (version.Equal(left) || version.Equal(right)) ||
			version.GreaterThan(left) && version.LessThan(right)
	case start == '[' && end == ')':
		return version.Equal(left) ||
			(lr[1] == "" && version.GreaterThan(left)) ||
			(version.GreaterThan(left) && version.LessThan(right))
	case start == '(' && end == ']':
		return version.Equal(right) ||
			(version.GreaterThan(left) && version.LessThan(right))
	case start == '(' && end == ')':
		return (lr[1] == "" && version.GreaterThan(left)) ||
			(version.GreaterThan(left) && version.LessThan(right))
	}

	return false
}
