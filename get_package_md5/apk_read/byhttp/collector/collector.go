package collector

import (
	"bufio"
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
}

func NewCollector(httpCli http.Client, tool string, sleep int) *Collector {
	return &Collector{httpCli: httpCli, tool: tool, sleepDelay: sleep}
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
	if err := c.processHTml(url, resp); err != nil {
		return errors.WithMessagef(err, "process %s", url)
	}
	return nil
}

func (c *Collector) processHTml(url string, resp *http.Response) error {
	if resp.Body == nil {
		return errors.New("nil resp body")
	}
	if resp.StatusCode != 200 {
		if resp.StatusCode == 429 || resp.StatusCode == 403 ||
			resp.StatusCode == 503 || resp.StatusCode == 504 {
			os.Exit(0)
		}
		return errors.Errorf("status code %d", resp.StatusCode)
	}

	return c.walkHtmlNode(resp.Body, func(href string) error {
		nextUrl := strings.TrimSuffix(url, "/") + "/" + href
		if strings.HasSuffix(href, ".apk") {
			if err := c.downloadByTool(nextUrl); err != nil {
				log.Println(err)
			}
		} else if strings.HasSuffix(href, "/") {
			if err := c.Visit(nextUrl); err != nil {
				log.Println(err)
			}
		}
		return nil
	})
}

func (c *Collector) walkHtmlNode(r io.Reader, do func(href string) error) error {
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

	log.Printf("success download %s\n", uu.String())
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

func sleep(t int) {
	rand.Seed(time.Now().UnixNano())
	x := rand.Intn(t)
	d := time.Millisecond * 100
	for i := 0; i < x; i++ {
		d += time.Millisecond * 100
	}
	time.Sleep(d)
}
