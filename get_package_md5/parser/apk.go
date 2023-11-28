package parser

import (
	"bufio"
	"get_package_md5/model"
	"get_package_md5/unarchiver"
	"get_package_md5/utils"
	errors2 "github.com/pkg/errors"
	"io"
	"strings"
)

type Apk struct{}

func NewApkParser() *Apk {
	return &Apk{}
}

func (p *Apk) Parse(i io.Reader, out string) error {
	pkg := new(model.ApkPkg)
	if err := unarchiver.ReadTarGzip(i, func(n string, r io.Reader) error {
		if isPkgInfo(n) {
			err := parsePkgInfo(r, pkg)
			if err != nil {
				return errors2.WithMessagef(err, "parse pkginfo")
			}
		}
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		if utils.NoBinary(n) {
			return nil
		}
		ok, val, e := utils.CheckElf(data)
		if e != nil {
			return errors2.WithMessagef(e, "calculate hash")
		}
		if !ok {
			return nil
		}
		pkg.Hashes = append(pkg.Hashes, model.Hash{
			Key:   val,
			Value: n,
		})
		return nil
	}); err != nil {
		return err
	}
	if err := utils.SaveJson(pkg, out); err != nil {
		return errors2.WithMessagef(err, "save json")
	}
	return nil
}

func (p *Apk) Check(n string) bool {
	return strings.HasSuffix(n, ".apk")
}

func isPkgInfo(n string) bool {
	return strings.HasSuffix(n, ".PKGINFO")
}

func parsePkgInfo(r io.Reader, pkg *model.ApkPkg) error {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "#") {
			continue
		}
		line := strings.TrimSpace(sc.Text())
		last := ""
		if strings.Contains(line, "=") {
			kvs := strings.SplitN(line, "=", 2)
			val := kvs[1]
			switch strings.TrimSpace(kvs[0]) {
			case "pkgname":
				pkg.PkgName = val
				last = "pkgname"
			case "pkgver":
				pkg.PkgVer = val
				last = "pkgver"
			case "pkgdesc":
				pkg.PkgDesc = val
				last = "pkgdesc"
			case "url":
				pkg.URL = val
				last = "url"
			case "packager":
				pkg.Packager = val
				last = "packager"
			case "arch":
				pkg.Arch = val
				last = "arch"
			case "maintainer":
				pkg.Maintainer = val
				last = "maintainer"
			case "license":
				pkg.License = append(pkg.License, val)
				last = "license"
			case "depend":
				pkg.Depend = append(pkg.Depend, val)
				last = "depend"
			case "builddate":
				pkg.BuildDate = val
				last = "builddate"
			case "size":
				pkg.Size = val
				last = "size"
			case "origin":
				pkg.Origin = val
				last = "origin"
			}
		} else {
			if last != "pkgdesc" {
				continue
			}
			pkg.PkgDesc = pkg.PkgDesc + " " + line
		}
	}
	return nil
}
