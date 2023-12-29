package es

import (
	"context"
	"fmt"
	"get_package_md5/model"

	"github.com/olivere/elastic/v7"
)

const Index = "pkg_bin_hash_final"

type Cli struct {
	cli *elastic.Client
}

func NewEsCli(addr string) (*Cli, error) {
	cli, err := elastic.NewClient(elastic.SetURL(addr), elastic.SetSniff(false))
	if err != nil {
		return nil, err
	}

	return &Cli{
		cli: cli,
	}, nil
}

func (es *Cli) Insert(docs []model.Document) error {
	bulkRequest := es.cli.Bulk()
	for _, doc := range docs {
		req := elastic.NewBulkIndexRequest().Index(Index).Doc(doc)
		bulkRequest = bulkRequest.Add(req)
	}

	bulkResponse, err := bulkRequest.Do(context.Background())
	if err != nil {
		return err
	}

	if bulkResponse == nil {
		return fmt.Errorf("nil response")
	}

	return nil
}
