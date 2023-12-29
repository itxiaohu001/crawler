package parser

import (
	"bufio"
	"fmt"
	"get_package_md5/model"
	"get_package_md5/unarchiver"
	"get_package_md5/utils"
	"io"
	"strings"

	errors2 "github.com/pkg/errors"
)

type Deb struct{}

func NewDebParser() *Deb {
	return &Deb{}
}

func (d *Deb) Parse(r io.Reader, out string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors2.New("panic err")
		}
	}()

	pkg := new(model.DebPkg)
	if err := unarchiver.ReadAr(r, func(n string, r io.Reader) error {
		if isControl(n) {
			return parseControl(n, r, pkg)
		} else if isData(n) {
			return parseData(n, r, pkg)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := utils.SaveJson(pkg, out); err != nil {
		return errors2.WithMessagef(err, "save json")
	}
	return nil
}

func (d *Deb) Check(n string) bool {
	return strings.HasSuffix(n, ".deb") ||
		strings.HasSuffix(n, ".udeb")
}

func parseControl(name string, r io.Reader, pkg *model.DebPkg) error {
	var readFunc unarchiver.ReadFunc

	if strings.HasSuffix(name, "control.tar.zst") {
		readFunc = unarchiver.ReadTarZst
	} else if strings.HasSuffix(name, "control.tar.xz") {
		readFunc = unarchiver.ReadTarXz
	} else if strings.HasSuffix(name, "control.tar.gz") {
		readFunc = unarchiver.ReadTarGzip
	} else if strings.HasSuffix(name, "control.tar") {
		readFunc = unarchiver.ReadTar
	} else if strings.HasSuffix(name, "control.tar.bz2") {
		readFunc = unarchiver.ReadTarBz2
	} else if strings.HasSuffix(name, "control.tar.lzma") {
		readFunc = unarchiver.ReadTarLzma
	} else {
		return fmt.Errorf("unexpected format %s", name)
	}

	return readFunc(r, func(n string, r io.Reader) error {
		return analyzeControlFile(n, r, pkg)
	})
}

func parseData(name string, r io.Reader, pkg *model.DebPkg) error {
	var readFunc unarchiver.ReadFunc

	if strings.HasSuffix(name, "data.tar.zst") {
		readFunc = unarchiver.ReadTarZst
	} else if strings.HasSuffix(name, "data.tar.xz") {
		readFunc = unarchiver.ReadTarXz
	} else if strings.HasSuffix(name, "data.tar.gz") {
		readFunc = unarchiver.ReadTarGzip
	} else if strings.HasSuffix(name, "data.tar") {
		readFunc = unarchiver.ReadTar
	} else if strings.HasSuffix(name, "data.tar.bz2") {
		readFunc = unarchiver.ReadTarBz2
	} else if strings.HasSuffix(name, "data.tar.lzma") {
		readFunc = unarchiver.ReadTarLzma
	} else {
		return fmt.Errorf("unexpected format %s", name)
	}

	return readFunc(r, func(n string, r io.Reader) error {
		return analyzeDataFile(n, r, pkg)
	})
}

func analyzeControlFile(name string, r io.Reader, p *model.DebPkg) error {
	if strings.Contains(name, "control") {
		scanner := bufio.NewScanner(r)
		dsValue := ""
		for scanner.Scan() {
			if dsValue != "" {
				dsValue = dsValue + scanner.Text()
				continue
			}
			i := strings.Index(scanner.Text(), ":")
			if i == -1 {
				continue
			}
			k := strings.TrimSpace(scanner.Text()[:i])
			v := strings.TrimSpace(scanner.Text()[i+1:])
			switch k {
			case "Package":
				p.Name = v
			case "Source":
				p.Source = v
			case "Version":
				p.Version = v
			case "Architecture":
				p.Architecture = v
			case "Maintainer":
				p.Maintainer = v
			case "Original-Maintainer":
				p.OriginalMaintainer = v
			case "Installed-Size":
				p.InstalledSize = v
			case "Depends":
				for _, dep := range strings.Split(v, ",") {
					p.Depends = append(p.Depends, strings.TrimSpace(dep))
				}
			case "Suggests":
				for _, dep := range strings.Split(v, ",") {
					p.Suggests = append(p.Suggests, strings.TrimSpace(dep))
				}
			case "Section":
				p.Section = v
			case "Priority":
				p.Priority = v
			case "Homepage":
				p.Homepage = v
			}
			if k == "Description" {
				// description信息有多行，需特殊处理
				dsValue = dsValue + v
			}
		}
		if dsValue != "" {
			p.Description = dsValue
		}
	}

	return nil
}

// analyzeDataFile 解析data文件，获取elf文件hash值
func analyzeDataFile(n string, r io.Reader, p *model.DebPkg) error {
	n = strings.TrimPrefix(n, "./")
	data, err := io.ReadAll(r)
	if err != nil {
		return errors2.Wrapf(err, "read %s", n)
	}
	l, _ := utils.CheckLicense(n, data)
	if l != nil {
		p.Licences = append(p.Licences, l)
		return nil
	}

	if p.Hashes == nil {
		p.Hashes = map[string]string{}
	}
	if utils.NoBinary(n) {
		return nil
	}
	if ok, md5Val, err := utils.CheckElf(data); ok {
		p.Hashes[md5Val] = n
	} else if err != nil {
		return errors2.Wrapf(err, "check reader")
	}
	return nil
}

func isControl(n string) bool {
	return strings.Contains(n, "control")
}

func isData(n string) bool {
	return strings.Contains(n, "data")
}
