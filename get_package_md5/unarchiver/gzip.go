package unarchiver

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"io"

	"github.com/blakesmith/ar"
	"github.com/cavaliergopher/cpio"
	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
	"github.com/ulikunitz/xz/lzma"
)

type ReadFunc func(r io.Reader, do func(n string, r io.Reader) error) error

func ReadAr(r io.Reader, do func(n string, r io.Reader) error) error {
	a := ar.NewReader(r)
	for {
		h, e := a.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return e
		}
		if err := do(h.Name, a); err != nil {
			return err
		}
	}
	return nil
}

func ReadTar(r io.Reader, do func(n string, r io.Reader) error) error {
	t := tar.NewReader(r)
	for {
		h, e := t.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return e
		}
		if h.FileInfo().IsDir() {
			continue
		}
		if err := do(h.Name, t); err != nil {
			return err
		}
	}
	return nil
}

func ReadTarGzip(r io.Reader, do func(n string, r io.Reader) error) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	return ReadTar(gr, do)
}

func ReadTarLzma(r io.Reader, do func(n string, r io.Reader) error) error {
	l, err := lzma.NewReader(r)
	if err != nil {
		return err
	}
	return ReadTar(l, do)
}

func ReadTarBz2(r io.Reader, do func(n string, r io.Reader) error) error {
	b := bzip2.NewReader(r)
	return ReadTar(b, do)
}

func ReadTarXz(r io.Reader, do func(n string, r io.Reader) error) error {
	x, err := xz.NewReader(r)
	if err != nil {
		return err
	}
	return ReadTar(x, do)
}

func ReadTarZst(r io.Reader, do func(n string, r io.Reader) error) error {
	z, err := zstd.NewReader(r)
	if err != nil {
		return err
	}

	return ReadTar(z, do)
}

func ReadCpioXz(r io.Reader, do func(n string, r io.Reader) error) error {
	x, err := xz.NewReader(r)
	if err != nil {
		return err
	}

	c := cpio.NewReader(x)
	for {
		h, err := c.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if !h.FileInfo().Mode().IsRegular() {
			continue
		}
		if err := do(h.Name, c); err != nil {
			return err
		}
	}
	return nil
}
