package qurery

type Search interface {
	// SearchByFilePath 通过文件的实际路径查询
	SearchByFilePath(index string, filePaths ...string) (map[string][]Document, error)
	// SearchByHash 通过hash值查询
	SearchByHash(index string, hashes ...string) (map[string][]Document, error)
	// SearchPkgVuln 返回pkg对应的漏洞编号列表
	SearchPkgVuln(index string, pkms ...*PkgKeyMessage) (map[*PkgKeyMessage][]string, error)
}
