package main

import (
	"archive/tar"
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"goLearning/get_package_md5/model"
	"goLearning/get_package_md5/utils"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blakesmith/ar"
	"github.com/h2non/filetype"

	jsoniter "github.com/json-iterator/go"
	"github.com/klauspost/compress/zstd"
	errors2 "github.com/pkg/errors"
	"github.com/ulikunitz/xz"
	"github.com/ulikunitz/xz/lzma"
)

var (
	InputDir       = ""
	RootDir        = ""
	MaxConcurrency int
	logFile        = "error_log"
)

func init() {
	flag.StringVar(&RootDir, "o", "", "指定json输出目录")
	flag.StringVar(&InputDir, "i", "", "指定待检测文件夹")
	flag.IntVar(&MaxConcurrency, "m", 8, "指定最大协程数")
	flag.Parse()

	if RootDir == "" {
		log.Fatal("请指定json输出目录")
	}
	if InputDir == "" {
		log.Fatal("请指定待检测文件夹")
	}

	_, err := os.Stat(RootDir)
	if err != nil {
		if e := os.MkdirAll(RootDir, 0755); e != nil {
			log.Fatal(e)
		}
	}
}

func main() {
	if err := WalkDirForDebPkg(InputDir); err != nil {
		log.Println(err)
	}
}

// WalkDirForDebPkg 遍历文件夹，解析pkg
func WalkDirForDebPkg(dir string) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, MaxConcurrency)

	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(path string, d fs.DirEntry) {
			defer func() {
				wg.Done()
				<-semaphore
			}()
			if d.IsDir() {
				return
			}
			// 仅处理.deb .udeb等文件
			if !strings.HasSuffix(d.Name(), "deb") {
				return
			}
			pkg, err := readDeb(path)
			if err != nil {
				utils.RecordErrors(err, logFile)
				return
			}
			err = saveInto(pkg, filepath.Join(RootDir, makeNestDir(path)))
			if err != nil {
				utils.RecordErrors(err, logFile)
				return
			}
		}(path, d)

		return nil
	})

	wg.Wait()
	return nil
}

// analyzeControlFile 解析control文件，获取元数据信息
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

// analyzeDataFile 解析data文件，获取各文件hash值
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
	if ok, md5, err := checkElfPe(data); ok {
		p.Hashes[md5] = n
	} else if err != nil {
		return errors2.Wrapf(err, "check reader")
	}
	return nil
}

// checkElfPe 检查是否是可执行文件并返回内容便于计算hash值
func checkElfPe(data []byte) (bool, string, error) {
	buffer := make([]byte, 261)
	copy(buffer, data)
	t, err := filetype.Get(buffer)
	if err != nil {
		return false, "", errors2.Wrapf(err, "get file type")
	}
	if t.Extension == "elf" {
		return true, calculateMd5(data), nil // ELF 二进制文件
	}
	return false, "", nil
}

// makeNestDir 获取上三级目录
func makeNestDir(p string) string {
	dirInner := filepath.Dir(p)
	dirMid := filepath.Dir(dirInner)
	dirTop := filepath.Dir(dirMid)
	return filepath.Join(filepath.Base(dirTop), filepath.Base(dirMid), filepath.Base(dirInner))
}

// 保存单个pkg至单个json文件
func saveInto(p *model.DebPkg, path string) error {
	if p == nil {
		return errors2.Wrapf(fmt.Errorf("nil pkg"), "save error")
	}
	// 缺失关键字段则没必要保存
	if p.Name == "" || p.Version == "" || len(p.Hashes) == 0 {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		if err := os.MkdirAll(path, 0644); err != nil {
			return errors2.Wrapf(err, "save error")
		}
	}
	if p.Architecture == "" {
		path = filepath.Join(path, strings.Join([]string{p.Name, p.Version}, "_")+".json")
	} else {
		path = filepath.Join(path, strings.Join([]string{p.Name, p.Version, p.Architecture}, "_")+".json")
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return errors2.Wrapf(err, "save path:%s", path)
	}
	defer f.Close()
	if err := jsoniter.NewEncoder(f).Encode(p); err != nil {
		return errors2.Wrapf(err, "save path:%s", path)
	}

	return nil
}

// 读取.deb文件
func readDeb(p string) (*model.DebPkg, error) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("panic error:", e)
		}
	}()

	f, err := os.Open(p)
	if err != nil {
		return nil, errors2.Wrapf(err, "open file %s", p)
	}
	defer f.Close()

	a := ar.NewReader(f)
	pkg := new(model.DebPkg)
	if err := parseDebName(p, pkg); err != nil {
		return nil, err
	}
	for {
		h, e := a.Next()
		if e != nil {
			break
		}
		if strings.Contains(h.Name, "control") {
			if err := processControl(h.Name, a, pkg); err != nil {
				return pkg, errors2.Wrapf(err, "process control file by %s", pkg.OriginName)
			}
		} else if strings.Contains(h.Name, "data") {
			if err := processData(h.Name, a, pkg); err != nil {
				return pkg, errors2.Wrapf(err, "process data file by %s", pkg.OriginName)
			}
		}
	}
	return pkg, nil
}

// processControl 解压各个格式的control文件
func processControl(name string, r io.Reader, p *model.DebPkg) error {
	if strings.HasSuffix(name, "control.tar.zst") {
		pp, err := readZst(r, analyzeControlFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else if strings.HasSuffix(name, "control.tar.xz") {
		pp, err := readXz(r, analyzeControlFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else if strings.HasSuffix(name, "control.tar.gz") {
		pp, err := readGz(r, analyzeControlFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else if strings.HasSuffix(name, "control.tar") {
		pp, err := readTar(r, analyzeControlFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else if strings.HasSuffix(name, "control.tar.bz2") {
		pp, err := readBz2(r, analyzeControlFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else if strings.HasSuffix(name, "control.tar.lzma") {
		pp, err := readLzma(r, analyzeControlFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else {
		return fmt.Errorf("unexpected format %s", name)
		// 目前来看未发现其他文件
	}

	return nil
}

// processData 解压各个格式的data文件
func processData(name string, r io.Reader, p *model.DebPkg) error {
	if strings.HasSuffix(name, "data.tar.zst") {
		pp, err := readZst(r, analyzeDataFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else if strings.HasSuffix(name, "data.tar.xz") {
		pp, err := readXz(r, analyzeDataFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else if strings.HasSuffix(name, "data.tar.gz") {
		pp, err := readGz(r, analyzeDataFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else if strings.HasSuffix(name, "data.tar") {
		pp, err := readTar(r, analyzeDataFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else if strings.HasSuffix(name, "data.tar.bz2") {
		pp, err := readBz2(r, analyzeDataFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else if strings.HasSuffix(name, "data.tar.lzma") {
		pp, err := readLzma(r, analyzeDataFile)
		if err != nil {
			return err
		}
		p.Merge(pp)
	} else {
		return fmt.Errorf("unexpected format %s", name)
	}

	return nil
}

// readLzma 读取tar.lzma文件
func readLzma(r io.Reader, do func(n string, r io.Reader, p *model.DebPkg) error) (*model.DebPkg, error) {
	l, err := lzma.NewReader(r)
	if err != nil {
		return nil, err
	}
	return readTar(l, do)
}

// readBz2 读取tar.bz2文件
func readBz2(r io.Reader, do func(n string, r io.Reader, p *model.DebPkg) error) (*model.DebPkg, error) {
	b := bzip2.NewReader(r)
	return readTar(b, do)
}

// readGz 读取tar.gz文件
func readGz(r io.Reader, do func(n string, r io.Reader, p *model.DebPkg) error) (*model.DebPkg, error) {
	g, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return readTar(g, do)
}

// readXz 读取tar.xz文件
func readXz(r io.Reader, do func(n string, r io.Reader, p *model.DebPkg) error) (*model.DebPkg, error) {
	x, err := xz.NewReader(r)
	if err != nil {
		return nil, err
	}

	return readTar(x, do)
}

// readZst 读取tar.zst文件
func readZst(r io.Reader, do func(n string, r io.Reader, p *model.DebPkg) error) (*model.DebPkg, error) {
	z, e := zstd.NewReader(r)
	if e != nil {
		return nil, e
	}

	return readTar(z, do)
}

func readTar(r io.Reader, do func(n string, r io.Reader, p *model.DebPkg) error) (*model.DebPkg, error) {
	t := tar.NewReader(r)
	pkg := new(model.DebPkg)
	for {
		h, e := t.Next()
		if e != nil {
			break
		}
		if h.FileInfo().IsDir() {
			continue
		}
		if err := do(h.Name, t, pkg); err != nil {
			log.Println(err)
		}
	}

	return pkg, nil
}

// parseDebName 通过文件名获取packageName、version、arch等信息
func parseDebName(n string, p *model.DebPkg) error {
	n = filepath.Base(n)
	p.OriginName = n
	var parts []string
	i := strings.LastIndex(n, ".")
	if i > 0 {
		parts = strings.Split(n[:i], "_")
	}
	if len(parts) == 3 {
		p.Name = parts[0]
		p.Version = parts[1]
		p.Architecture = parts[2]
	} else if len(parts) == 2 {
		p.Name = parts[0]
		p.Version = parts[1]
	} else {
		return errors2.Wrapf(fmt.Errorf("unexpected deb name"), "parse deb name %s", n)
	}
	return nil
}

// calculateMd5 计算md5值
func calculateMd5(b []byte) string {
	hashValue := md5.Sum(b)
	md5Str := hex.EncodeToString(hashValue[:])
	return md5Str
}
