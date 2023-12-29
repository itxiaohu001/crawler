package parser

import (
	"fmt"
	"get_package_md5/model"
	"get_package_md5/unarchiver"
	"get_package_md5/utils"
	"io"
	"strings"

	"github.com/cavaliergopher/rpm"
	"github.com/pkg/errors"
)

type Rpm struct{}

func NewRpmParser() *Rpm {
	return &Rpm{}
}

func (r2 *Rpm) Parse(r io.Reader, out string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.Errorf("panic err %s", e)
		}
	}()

	rpmPkg := new(model.RpmPkg)
	o, _ := utils.Extract("rpm", out)
	if o != "" {
		rpmPkg.OS = o
	}

	pkg, err := rpm.Read(r)
	if err != nil {
		return errors.WithMessagef(err, "read rpm head")
	}
	assignTo(pkg, rpmPkg)

	if compression := pkg.PayloadCompression(); compression != "xz" {
		return errors.Errorf("cannot parse such compression of rpm %s", compression)
	}
	if err := unarchiver.ReadCpioXz(r, func(n string, r io.Reader) error {
		if utils.NoBinary(n) {
			return nil
		}

		data, err := io.ReadAll(r)
		if err != nil {
			return errors.WithMessagef(err, "read reader")
		}

		if ok, md5Val, err := utils.CheckElf(data); ok {
			rpmPkg.Hashes = append(rpmPkg.Hashes, model.Hash{Key: md5Val, Value: n})
		} else if err != nil {
			return errors.WithMessagef(err, "check elf")
		}

		return nil
	}); err != nil {
		return err
	}

	if err := utils.SaveJson(rpmPkg, out); err != nil {
		return errors.WithMessagef(err, "save json")
	}

	return nil
}

func (r2 *Rpm) Check(n string) bool {
	return strings.HasSuffix(n, ".rpm")
}

func assignTo(p *rpm.Package, pp *model.RpmPkg) {
	if p == nil {
		return
	}

	pp.Manager = "rpm"
	pp.Name = p.Name()
	pp.Version = p.Version()
	pp.Homepage = p.URL()
	for _, d := range p.Requires() {
		deb := fmt.Sprintf("%s;%s;%s;%d", d.Name(), d.Version(), d.Release(), d.Epoch())
		pp.Depends = append(pp.Depends, deb)
	}
	pp.Architecture = p.Architecture()
	pp.Description = p.Description()
	pp.License = append(pp.License, p.License())
	pp.Vendor = p.Vendor()
	pp.Release = p.Release()
	pp.Epoch = p.Epoch()
	pp.Source = p.SourceRPM()
	pp.Maintainer = p.Packager()
}
