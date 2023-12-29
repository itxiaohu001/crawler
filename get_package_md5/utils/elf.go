package utils

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"github.com/h2non/filetype"
	errors2 "github.com/pkg/errors"
)

var nonBinaryFileExtensions = []string{".txt", ".csv", ".xml", ".json", ".html", ".md", ".yml", ".yaml", ".ini", ".conf", ".bat", ".sh", ".py", ".java", ".cpp", ".js", ".go", ".h", ".c"}

func NoBinary(n string) bool {
	for _, suffix := range nonBinaryFileExtensions {
		if strings.HasSuffix(n, suffix) {
			return true
		}
	}
	return false
}

func CheckElf(data []byte) (bool, string, error) {
	buffer := make([]byte, 261)
	copy(buffer, data)
	t, err := filetype.Get(buffer)
	if err != nil {
		return false, "", errors2.WithMessagef(err, "get file type")
	}
	if t.Extension == "elf" {
		return true, CalculateMd5(data), nil // ELF 二进制文件
	}
	return false, "", nil
}

func CalculateMd5(b []byte) string {
	hashValue := md5.Sum(b)
	md5Str := hex.EncodeToString(hashValue[:])
	return md5Str
}
