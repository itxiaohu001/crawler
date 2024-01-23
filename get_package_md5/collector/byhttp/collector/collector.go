package collector

import (
	"bufio"
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
	parsers    []parser.Parser
	sleeper    *sleeper.Sleeper
}

func NewCollector(httpCli http.Client, parsers []parser.Parser, logger *recorder.AccessRecorder,
	pool *pool.Pool, sleeper *sleeper.Sleeper) *Collector {

	return &Collector{
		pool:       pool,
		HTTPClient: httpCli,
		sleeper:    sleeper,
		parsers:    parsers,
		Recorder:   logger,
	}
}

func (c *Collector) Wait() {
	c.pool.Wait()
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
		for _, p := range c.parsers {
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

	if c.Recorder.Exist(uu.String()) {
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
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	dir := "." + path.Dir(uu.Path)
	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0644); err != nil {
			return errors.WithMessagef(err, "create dir %s", dir)
		}
	}
	saveName := strings.TrimSuffix(filepath.Base(uu.Path), filepath.Ext(uu.Path)) + ".json"

	for _, p := range c.parsers {
		if p.Check(uu.Path) {
			if err := p.Parse(resp.Body, path.Join(dir, saveName)); err != nil {
				return errors.WithMessagef(err, "parse %s", uu.String())
			}
			break
		}
	}

	c.Recorder.Record(uu.String())
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
