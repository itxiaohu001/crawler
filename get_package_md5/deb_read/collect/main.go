package main

import (
	"archive/tar"
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"get_package_md5/model"
	"get_package_md5/utils"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"

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
)

const (
	_debSuffix         = "deb"
	_historyFile       = "history.log"
	_statisticSavePath = "statistics.log"
	_errorLog          = "error.log"
)

// todo:添加history文件
// todo:hash碰撞统计
// todo:分析程序被终止原因

type NumPaths struct {
	HashMd5 string
	Num     int
	Paths   []string
}

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
		if e := os.MkdirAll(RootDir, 0644); e != nil {
			log.Fatal(e)
		}
	}
}

type Parser struct {
	targetDir         string
	outDir            string
	historyFile       string
	statisticSavePath string
	errorLog          string
	historyInfo       map[string]struct{}
	statisticalInfo   map[string]NumPaths
	maxConcurrency    int
	wg                sync.WaitGroup
	mu                sync.Mutex
}

func NewParser(targetDir, outDir string, max int) (*Parser, error) {
	hi := make(map[string]struct{})

	log.Printf("\nload history...\n")
	f, err := os.OpenFile(_historyFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, errors2.WithMessagef(err, "open file %s", _historyFile)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		hi[strings.TrimSpace(sc.Text())] = struct{}{}
	}
	log.Printf("load history success,%d records in all", len(hi))

	return &Parser{
		targetDir:         targetDir,
		outDir:            outDir,
		maxConcurrency:    max,
		historyFile:       _historyFile,
		statisticSavePath: _statisticSavePath,
		errorLog:          _errorLog,
		historyInfo:       hi,
		statisticalInfo:   map[string]NumPaths{},
		wg:                sync.WaitGroup{},
		mu:                sync.Mutex{},
	}, nil
}

func main() {
	parser, _ := NewParser(InputDir, RootDir, MaxConcurrency)
	defer func() {
		if err := parser.saveStatisticalInfo(); err != nil {
			log.Fatal(err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)
	go func() {
		<-sigs
		if parser != nil {
			parser.saveStatisticalInfo()
		}
		os.Exit(0)
	}()

	if err := parser.WalkDirForDebPkg(); err != nil {
		log.Fatal(err)
	}
}

// WalkDirForDebPkg 遍历文件夹，解析pkg
func (p *Parser) WalkDirForDebPkg() error {
	semaphore := make(chan struct{}, p.maxConcurrency)

	filepath.WalkDir(p.targetDir, func(path string, d fs.DirEntry, err error) error {
		p.wg.Add(1)
		semaphore <- struct{}{}

		go func(path string, d fs.DirEntry) {
			defer func() {
				p.wg.Done()
				<-semaphore
			}()

			if _, ok := p.historyInfo[path]; ok {
				return
			}
			if d.IsDir() {
				return
			}
			if !strings.HasSuffix(d.Name(), _debSuffix) {
				return
			}

			if err := p.ReadDeb(path); err != nil {
				p.recordError(errors2.WithMessagef(err, "read %s", path))
				return
			}
		}(path, d)

		return nil
	})

	p.wg.Wait()
	return nil
}

func (p *Parser) record(pkg *model.DebPkg, fp string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if pkg == nil {
		return nil
	}

	f, err := os.OpenFile(p.historyFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return errors2.WithMessagef(err, "open file %s", p.historyFile)
	}
	defer f.Close()
	_, _ = f.WriteString(fp + "\n")

	for k, v := range pkg.Hashes {
		np := p.statisticalInfo[k]
		np.HashMd5 = k
		np.Num++
		np.Paths = append(np.Paths, fp+"_"+v)
		p.statisticalInfo[k] = np
	}

	return nil
}

func (p *Parser) recordError(err error) error {
	if err == nil {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	f, e := os.OpenFile(p.errorLog, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if e != nil {
		return errors2.WithMessagef(err, "open file %s", p.errorLog)
	}
	defer f.Close()

	f.WriteString(err.Error() + "\n")
	return nil
}

func (p *Parser) saveStatisticalInfo() error {
	infos := make([]NumPaths, len(p.statisticalInfo))
	i := 0
	for _, v := range p.statisticalInfo {
		infos[i] = v
		i++
	}
	slices.SortFunc(infos, func(a, b NumPaths) int {
		return a.Num - b.Num
	})

	f, err := os.OpenFile(p.statisticSavePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		errors2.WithMessagef(err, "open file %s", p.statisticSavePath)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(infos[:len(infos)/1000]); err != nil {
		return errors2.WithMessagef(err, "encode statistics data")
	}

	return nil
}

func (p *Parser) ReadDeb(deb string) error {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("panic error:", e)
		}
	}()

	f, err := os.Open(deb)
	if err != nil {
		return errors2.Wrapf(err, "open file %s", deb)
	}
	defer f.Close()

	a := ar.NewReader(f)
	pkg := new(model.DebPkg)
	if err := parseDebName(deb, pkg); err != nil {
		return err
	}
	for {
		h, e := a.Next()
		if e != nil {
			break
		}
		if strings.Contains(h.Name, "control") {
			if err := processControl(h.Name, a, pkg); err != nil {
				return errors2.Wrapf(err, "process control file by %s", pkg.OriginName)
			}
		} else if strings.Contains(h.Name, "data") {
			if err := processData(h.Name, a, pkg); err != nil {
				return errors2.Wrapf(err, "process data file by %s", pkg.OriginName)
			}
		}
	}

	p.record(pkg, deb)

	return saveInto(pkg, filepath.Join(p.outDir, makeNestDir(deb)))
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
	if ok, md5Val, err := checkElf(data); ok {
		p.Hashes[md5Val] = n
	} else if err != nil {
		return errors2.Wrapf(err, "check reader")
	}
	return nil
}

// checkElf 检查是否是可执行文件并返回内容便于计算hash值
func checkElf(data []byte) (bool, string, error) {
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
	if p.Name == "" || p.Version == "" || len(p.Hashes) == 0 {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		if err := os.MkdirAll(path, 0644); err != nil {
			return errors2.Wrapf(err, "mkdir %s", path)
		}
	}
	if p.Architecture == "" {
		path = filepath.Join(path, strings.Join([]string{p.Name, p.Version}, "_")+".json")
	} else {
		path = filepath.Join(path, strings.Join([]string{p.Name, p.Version, p.Architecture}, "_")+".json")
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors2.Wrapf(err, "save path:%s", path)
	}
	defer f.Close()

	if err := jsoniter.NewEncoder(f).Encode(p); err != nil {
		return errors2.Wrapf(err, "save path:%s", path)
	}

	return nil
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
