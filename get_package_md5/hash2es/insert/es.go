package insert

import (
	"context"
	"fmt"
	"get_package_md5/model"

	"github.com/olivere/elastic/v7"
)

// Index 指定索引为"pkg_bin_hash_final"
const Index = "pkg_bin_hash_final"

type EsCli struct {
	cli *elastic.Client
}

func NewEsCli(addr string) (*EsCli, error) {
	cli, err := elastic.NewClient(elastic.SetURL(addr), elastic.SetSniff(false))
	if err != nil {
		return nil, err
	}

	return &EsCli{
		cli: cli,
	}, nil
}

func (es *EsCli) Insert(docs []model.Document) error {
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
