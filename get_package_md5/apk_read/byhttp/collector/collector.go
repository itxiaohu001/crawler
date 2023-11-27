package collector

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type Collector struct {
	httpCli         http.Client
	tool            string
	sleepDelay      int
	historyFilePath string
	errorFilePath   string
	historyMap      map[string]struct{}
	suffix          string
}

const (
	_historyFilePath = "download_history.log"
	_errorFilePath   = "error.log"
)

func NewCollector(httpCli http.Client, suffix string, tool string, sleep int) *Collector {
	f, err := os.OpenFile(_historyFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	history := map[string]struct{}{}
	sc := bufio.NewScanner(f)
	var counter int
	log.Printf("Loading download history...\n")
	for sc.Scan() {
		counter++
		history[strings.TrimSpace(sc.Text())] = struct{}{}
	}
	log.Printf("Loading download history success,%d records in all\n", counter)

	return &Collector{
		httpCli:         httpCli,
		tool:            tool,
		sleepDelay:      sleep,
		historyFilePath: _historyFilePath,
		errorFilePath:   _errorFilePath,
		suffix:          suffix,
		historyMap:      history,
	}
}

func (c *Collector) Visit(url string) error {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return errors.WithMessagef(err, "create request for %s", url)
	}
	resp, err := c.httpCli.Do(request)
	if err != nil {
		return errors.WithMessagef(err, "visit %s", url)
	}
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()
	if err := c.readHtml(url, resp); err != nil {
		return errors.WithMessagef(err, "process %s", url)
	}
	return nil
}

func (c *Collector) readHtml(url string, resp *http.Response) error {
	if resp.Body == nil {
		return errors.New("nil resp body")
	}
	if resp.StatusCode != 200 {
		if isBanCode(resp.StatusCode) {
			_ = c.recordError(fmt.Errorf("access has been blocked"))
			os.Exit(0)
		}
		return errors.Errorf("status code %d", resp.StatusCode)
	}

	return c.iterateAllHrefs(resp.Body, func(href string) error {
		nextUrl := strings.TrimSuffix(url, "/") + "/" + href
		if isTargetHref(href, c.suffix) {
			if err := c.downloadByTool(nextUrl); err != nil {
				_ = c.recordError(err)
			}
		} else if isDirHref(href) {
			if err := c.Visit(nextUrl); err != nil {
				_ = c.recordError(err)
			}
		}
		return nil
	})
}

func (c *Collector) iterateAllHrefs(r io.Reader, do func(href string) error) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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

func (c *Collector) downloadByTool(u string) error {
	uu, err := url.Parse(u)
	if err != nil {
		return errors.WithMessagef(err, "parse %s", u)
	}
	// 跳过下载过的文件
	if _, ok := c.historyMap[uu.String()]; ok {
		return nil
	}

	dir := "." + path.Dir(uu.Path)
	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0644); err != nil {
			return errors.WithMessagef(err, "create dir %s", dir)
		}
	}

	params := []string{"-P", dir, uu.String()}
	cmd := exec.Command(c.tool, params...)
	if err := cmd.Run(); err != nil {
		return errors.WithMessagef(err, "download %s by %s", uu.String(), c.tool)
	}

	if err := c.record(uu.String()); err != nil {
		log.Fatal(err)
	}
	sleep(c.sleepDelay)

	return nil
}

func (c *Collector) record(s string) error {
	s = strings.TrimSpace(s) + "\n"
	f, err := os.OpenFile(c.historyFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return errors.WithMessagef(err, "open %s", c.historyFilePath)
	}
	defer f.Close()
	if _, err := f.WriteString(s); err != nil {
		return errors.WithMessagef(err, "write string to %s", c.historyFilePath)
	}
	return nil
}

func (c *Collector) recordError(err error) error {
	if err == nil {
		return nil
	}
	errStr := strings.TrimSpace(err.Error()) + "\n"

	f, err := os.OpenFile(c.errorFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return errors.WithMessagef(err, "open %s", c.historyFilePath)
	}
	defer f.Close()

	if _, err := f.WriteString(errStr); err != nil {
		return errors.WithMessagef(err, "write string to %s", c.errorFilePath)
	}
	return nil
}

func isTargetHref(p, suffix string) bool {
	return strings.HasSuffix(p, suffix)
}

func isDirHref(p string) bool {
	return strings.HasSuffix(p, "/")
}

func isBanCode(code int) bool {
	return code == 429 || code == 403 ||
		code == 503 || code == 504
}

func sleep(t int) {
	rand.Seed(time.Now().UnixNano())
	x := rand.Intn(t)
	d := time.Millisecond * 100
	for i := 0; i < x; i++ {
		d += time.Millisecond * 100
	}
	time.Sleep(d)
}
