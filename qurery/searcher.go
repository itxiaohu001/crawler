package qurery

import (
	esv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/pkg/errors"
)

type Option func(searcher *Searcher) error

type Version int

const (
	Es7 Version = 7
	Es8 Version = 8
)

type Config struct {
	Version  Version
	Address  []string
	Username string
	Password string
}

func WithConfig(c *Config) func(searcher *Searcher) error {
	return func(searcher *Searcher) error {
		if c == nil {
			return errors.Errorf("empty config")
		}
		search, err := newSearch(c)
		if err != nil {
			return err
		}
		searcher.Search = search
		return nil
	}
}

type Searcher struct {
	Search
	c *Config
}

func NewSearcher(option ...Option) (*Searcher, error) {
	s := new(Searcher)
	for _, opt := range option {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func newSearch(c *Config) (Search, error) {
	switch c.Version {
	default:
		cli, err := esv7.NewClient(esv7.Config{
			Addresses: c.Address,
			Username:  c.Username,
			Password:  c.Password,
		})
		if err != nil {
			return nil, err
		}
		search := new(v7)
		search.cli = cli
		return search, nil
	}
}
