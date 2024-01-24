package collector

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"get_package_md5/collector/byhttp/recorder"
	"get_package_md5/collector/byhttp/sleeper"
	"get_package_md5/parser"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sourcegraph/conc/pool"
)

type Collector struct {
	HTTPClient http.Client
	Recorder   *recorder.AccessRecorder
	pool       *pool.Pool
	sleeper    *sleeper.Sleeper
	outDir     string
}

func NewCollector(httpCli http.Client, recorder *recorder.AccessRecorder,
	pool *pool.Pool, sleeper *sleeper.Sleeper, out string) *Collector {

	return &Collector{
		pool:       pool,
		HTTPClient: httpCli,
		sleeper:    sleeper,
		Recorder:   recorder,
		outDir:     out,
	}
}

func (c *Collector) Wait() {
	c.pool.Wait()
}

// var host = "https://mirrors.ustc.edu.cn/ubuntu/pool/universe/"
// var list = []string{"o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
// var host = "https://mirrors.ustc.edu.cn/alpine/"
// var list = []string{"v3.9", "v3.10", "v3.11", "v3.12", "v3.13", "v3.14", "v3.15", "v3.16", "v3.17", "v3.18", "v3.19"}

func (c *Collector) Start(do func(string)) {
	for pkgType, host := range hostRegisters {
		log.Printf("start to collecting %s packages from %s\n", pkgType, host)
		err := c.Visit(host, do)
		if err != nil {
			log.Println(err)
		}
	}
}

func (c *Collector) Visit(url string, do func(nextUrl string)) error {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return errors.WithMessagef(err, "create request for %s", url)
	}

	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return errors.WithMessagef(err, "visit %s", url)
	}

	if resp != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != 200 {
		if isBanCode(resp.StatusCode) {
			c.Recorder.RecordError("Access denied")
			os.Exit(0)
		}
		return errors.Errorf("status code %d", resp.StatusCode)
	}

	if err := c.walkResp(url, resp, do); err != nil {
		return errors.WithMessagef(err, "process %s", url)
	}

	return nil
}

func (c *Collector) walkResp(root string, resp *http.Response, do func(string)) error {
	return c.iterateHrefs(resp.Body, func(href string) error {
		nextUrl := strings.TrimSuffix(root, "/") + "/" + href
		if isDirHref(href) {
			if err := c.Visit(nextUrl, do); err != nil {
				c.Recorder.RecordError(err.Error())
			}
		}
		for _, p := range parserRegisters {
			if !p.Check(href) {
				continue
			}
			c.pool.Go(func() {
				do(nextUrl)
			})
			break
		}
		return nil
	})
}

func (c *Collector) iterateHrefs(r io.Reader, do func(string) error) error {
	var (
		lines   []string
		scanner = bufio.NewScanner(r)
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lines = append(lines, line)
	}

	for _, line := range lines {
		if strings.HasPrefix(line, `<a href="`) {
			left := strings.Index(line, `="`) + 2
			right := strings.Index(line, `">`)
			if left > right {
				log.Println("parse error", line)
				continue
			}
			if err := do(line[left:right]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Collector) DownloadAndParse(u string) error {
	uu, err := url.Parse(u)
	if err != nil {
		return errors.WithMessagef(err, "parse %s", u)
	}

	if c.Recorder.Exist(uu.Path) {
		log.Printf("skip downloaded url %s\n", uu.String())
		return nil
	}
	log.Printf("downloading %s\n", uu.String())

	// c.sleeper.Sleep()

	req, err := http.NewRequest("GET", uu.String(), nil)
	if err != nil {
		return errors.WithMessagef(err, "download %s", uu.String())
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return errors.WithMessagef(err, "download %s", uu.String())
	}
	if resp != nil {
		defer resp.Body.Close()
	}

	dir := filepath.Join(c.outDir, uu.Path)
	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0644); err != nil {
			return errors.WithMessagef(err, "create dir %s", dir)
		}
	}
	saveName := strings.TrimSuffix(filepath.Base(uu.Path), filepath.Ext(uu.Path)) + ".json"

	for _, p := range parserRegisters {
		if p.Check(uu.Path) {
			if err := p.Parse(resp.Body, path.Join(dir, saveName)); err != nil {
				return errors.WithMessagef(err, "parse %s", uu.String())
			}
			break
		}
	}

	c.Recorder.Record(uu.Path)
	log.Printf("download %s success\n", uu.String())

	return nil
}

func isDirHref(p string) bool {
	return strings.HasSuffix(p, "/")
}

func isBanCode(code int) bool {
	return code == 429 || code == 403 ||
		code == 503 || code == 504
}

func Register(pkgTypes []string) {
	for _, pkgType := range pkgTypes {
		switch pkgType {
		case "ubuntu":
			RegisterUbuntu()
		case "debian":
			RegisterDebian()
		case "alpine":
			RegisterAlpine()
		case "centos":
			RegisterRpm()
		default:
			log.Printf("invalid pkg type %s\n", pkgType)
		}
	}
}

var hostRegisters = map[string]string{}
var parserRegisters = map[string]parser.Parser{}

func RegisterUbuntu() {
	hostRegisters["ubuntu"] = "https://mirrors.ustc.edu.cn/ubuntu/pool"
	parserRegisters["ubuntu"] = parser.NewDebParser()
}

func RegisterAlpine() {
	hostRegisters["alpine"] = "https://mirrors.ustc.edu.cn/alpine/"
	parserRegisters["alpine"] = parser.NewApkParser()
}

func RegisterRpm() {
	hostRegisters["rpm"] = "https://mirrors.ustc.edu.cn/centos/"
	parserRegisters["rpm"] = parser.NewRpmParser()
}

func RegisterDebian() {
	hostRegisters["debian"] = "https://mirrors.ustc.edu.cn/debian/pool"
	parserRegisters["debian"] = parser.NewDebParser()
}

func (c *Collector) visitDeb(pkgType string, do func(string)) error {
	log.Printf("getting remote deb url list...\n")
	urls, err := GetDebFileList(pkgType)
	log.Printf("getting url list success,total %d\n", len(urls))
	if err != nil {
		return err
	}
	for _, u := range urls {
		u := u
		c.pool.Go(func() {
			do(u)
		})
	}
	return nil
}

func GetDebFileList(pkgType string) ([]string, error) {
	u := ""
	if pkgType == "ubuntu" {
		u = "https://mirrors.ustc.edu.cn/ubuntu/ubuntu/ls-lR.gz"
	} else if pkgType == "debian" {
		u = "https://mirrors.ustc.edu.cn/debian/ls-lR.gz"
	} else {
		return nil, fmt.Errorf("invalid pkg type %s", pkgType)
	}

	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		defer resp.Body.Close()
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, nil
	}

	data, err := io.ReadAll(gr)
	if err != nil {
		return nil, err
	}

	var allFiles []string
	parts := strings.Split(string(data), "\n\n")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		lines := strings.Split(part, "\n")
		if len(lines) < 3 {
			continue
		}
		d := strings.TrimPrefix(strings.TrimSuffix(lines[0], ":"), "./")
		if !strings.HasPrefix(d, "pool") {
			continue
		}
		for _, line := range lines {
			if strings.HasSuffix(line, ".deb") ||
				strings.HasSuffix(line, ".udeb") {
				pls := strings.Split(line, " ")
				allFiles = append(allFiles, hostRegisters[pkgType]+d+"/"+strings.TrimSpace(pls[len(pls)-1]))
			}
		}
	}

	return allFiles, nil
}
