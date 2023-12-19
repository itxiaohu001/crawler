package main

// Document 定义了Elasticsearch索引的结构
type Document struct {
	Manager            string   `json:"manager"`
	Name               string   `json:"name"`
	Source             string   `json:"source"`
	License            []string `json:"license"` // 现在是字符串数组
	Version            string   `json:"version"`
	Architecture       string   `json:"architecture"`
	Maintainer         string   `json:"maintainer"`
	OriginalMaintainer string   `json:"originalMaintainer"`
	Homepage           string   `json:"homepage"`
	Description        string   `json:"description"`
	Depends            string   `json:"depends"`
	Hashes             []Hash   `json:"hashes"`
}

// Hash 代表hashes字段中的结构
type Hash struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
