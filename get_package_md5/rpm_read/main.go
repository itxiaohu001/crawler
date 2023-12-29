package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/cavaliergopher/cpio"
	"github.com/ulikunitz/xz"

	"github.com/cavaliergopher/rpm"
	"github.com/h2non/filetype"
	errors2 "github.com/pkg/errors"
)

func main() {
	ExtractRPM("E:\\tmp\\test\\docker-unit-test-1.10.3-59.1.el7.x86_64.rpm")
}

func ExtractRPM(name string) {
	// Open a package file for reading
	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Read the package headers
	start := time.Now()
	pkg, err := rpm.Read(f)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("name %s\n", pkg.Name())
	fmt.Printf("version %s\n", pkg.Version())
	fmt.Printf("size %d\n", pkg.Size())
	fmt.Printf("license %s\n", pkg.License())
	fmt.Printf("packager %s\n", pkg.Packager())
	fmt.Printf("url %s\n", pkg.URL())
	fmt.Printf("arch %s\n", pkg.Architecture())
	fmt.Printf("desc %s\n", pkg.Description())
	fmt.Printf("src %s\n", pkg.Source())
	fmt.Printf("src rpm %s\n", pkg.SourceRPM())
	fmt.Printf("release %s\n", pkg.Release())
	fmt.Printf("epoch %d\n", pkg.Epoch())
	fmt.Printf("vendor %s\n", pkg.Vendor())
	for _, req := range pkg.Requires() {
		fmt.Println(req.Name())
	}

	//Check the compression algorithm of the payload
	if compression := pkg.PayloadCompression(); compression != "xz" {
		log.Fatalf("Unsupported compression: %s", compression)
	}

	// Attach a reader to decompress the payload
	xzReader, err := xz.NewReader(f)
	if err != nil {
		log.Fatal(err)
	}

	// Check the archive format of the payload
	if format := pkg.PayloadFormat(); format != "cpio" {
		log.Fatalf("Unsupported payload format: %s", format)
	}

	// Attach a reader to unarchive each file in the payload
	cpioReader := cpio.NewReader(xzReader)
	for {
		// Move to the next file in the archive
		hdr, err := cpioReader.Next()
		if err == io.EOF {
			break // no more files
		}
		if err != nil {
			log.Fatal(err)
		}

		// Skip directories and other irregular file types in this example
		if !hdr.FileInfo().Mode().IsRegular() {
			continue
		}

		//if utils.NoBinary(hdr.Name) {
		//	continue
		//}
		if ok, m, err := check(cpioReader); ok {
			fmt.Printf("file:%s, md5:%s\n", hdr.Name, m)
		} else if err != nil {
			fmt.Println(err)
		}
	}
	fmt.Println("cost time", time.Since(start).String())
}

func check(r io.Reader) (bool, string, error) {
	buffer := make([]byte, 261)
	_, err := r.Read(buffer)
	if err != nil && err != io.EOF {
		return false, "", errors2.Wrapf(err, "read buffer")
	}
	t, err := filetype.Get(buffer)
	if err != nil {
		return false, "", errors2.Wrapf(err, "get file type")
	}
	if t.Extension == "elf" {
		rest, err := io.ReadAll(r)
		if err != nil {
			return false, "", errors2.Wrapf(err, "read all")
		}
		return true, calculateMd5(append(buffer, rest...)), nil // ELF 二进制文件
	}
	return false, "", nil
}

func calculateMd5(b []byte) string {
	hashValue := md5.Sum(b)
	md5Str := hex.EncodeToString(hashValue[:])
	return md5Str
}
