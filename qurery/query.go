package qurery

type Search interface {
	// SearchByHash 通过单条hash值查询
	SearchByHash(h, index string) ([]Document, error)
	// MultiSearch 通过多条hash值批量查询多个结果
	MultiSearch(hs []string, index string) (map[string][]Document, error)
	// SearchByFilePath 通过文件的实际路径查询
	SearchByFilePath(p, index string) ([]Document, error)
	// MultiSearchByFilePath 通过多条文件的实际路径查询
	MultiSearchByFilePath(ps []string, index string) (map[string][]Document, error)
	// todo:添加其他查询字段
}
