package es

import (
	"context"
	"get_package_md5/model"
	"github.com/olivere/elastic/v7"
)

type Client struct {
	C *elastic.Client
}

func NewCli(url string) (*Client, error) {
	cli, err := elastic.NewClient(elastic.SetURL(url))
	if err != nil {
		return nil, err
	}
	return &Client{
		C: cli,
	}, nil
}

func (c *Client) Insert(pkg *model.DebPkg, index string) error {
	_, err := c.C.Index().Index(index).
		BodyJson(pkg).
		Do(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Search() (model.DebPkg, error) {
	return model.DebPkg{}, nil
}
