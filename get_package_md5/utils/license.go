package utils

import (
	"get_package_md5/model"
	"github.com/google/licensecheck"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Base names of guessable license files
	licenseFileNames = []string{
		"(?i)li[cs]en[cs]e(s?)",
		"(?i)legal",
		"(?i)copy(left|right|ing)",
		"(?i)unlicense",
		"(?i)l?gpl([-_ v]?)(\\d\\.?\\d)?",
		"(?i)bsd",
		"(?i)mit",
		"(?i)apache",
	}

	// License file extensions. Combined with the fileNames slice
	// to create a set of files we can reasonably assume contain
	// licensing information.
	fileExtensions = []string{
		"",
		".md",
		".rst",
		".html",
		".txt",
	}
)

func CheckLicense(n string, data []byte) (*model.License, error) {
	for i, ext := range fileExtensions {
		if strings.HasSuffix(n, ext) {
			break
		}
		if i == len(fileExtensions)-1 {
			return nil, nil
		}
	}
	for i, regStr := range licenseFileNames {
		reg := regexp.MustCompile(regStr)
		if reg.MatchString(filepath.Base(n)) {
			break
		}
		if i == len(licenseFileNames)-1 {
			return nil, nil
		}
	}

	lic := new(model.License)
	cov := licensecheck.Scan(data)
	lic.Per = cov.Percent
	lic.Path = n
	for _, m := range cov.Match {
		lic.Names = append(lic.Names, m.ID)
	}
	//licensecheck.BuiltinLicenses()
	if len(lic.Names) == 0 {
		return nil, nil
	}

	return lic, nil
}
