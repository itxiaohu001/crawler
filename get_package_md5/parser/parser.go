package parser

import "io"

type Parser interface {
	Parse(r io.Reader, out string) error
	Check(n string) bool
}
