package parser

import (
	"get_package_md5/unarchiver"
	"io"
	"strings"
)

type Rpm struct{}

func NewRpm() *Rpm {
	return &Rpm{}
}

func (r2 *Rpm) Parse(r io.Reader, out string) error {
	unarchiver.ReadCpioXz(r, func(n string, r io.Reader) error {

		return nil
	})
	return nil
}

func (r2 *Rpm) Check(n string) bool {
	return strings.HasSuffix(n, ".rpm")
}
