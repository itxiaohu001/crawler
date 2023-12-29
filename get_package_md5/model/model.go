package model

type Document struct {
	Os           string   `json:"os"`
	Epoch        int      `json:"epoch"`
	Release      string   `json:"release"`
	Manager      string   `json:"manager"`
	Name         string   `json:"name"`
	Source       string   `json:"source"`
	Version      string   `json:"version"`
	Architecture string   `json:"architecture"`
	Maintainer   string   `json:"maintainer"`
	Homepage     string   `json:"homepage"`
	Description  string   `json:"description"`
	License      []string `json:"license"`
	Depends      []string `json:"depends"`
	Hashes       []Hash   `json:"hashes"`
}

type Hash struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
