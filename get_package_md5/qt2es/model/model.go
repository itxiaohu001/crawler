package model

type Document struct {
	Manager            string   `json:"manager"`
	Name               string   `json:"name"`
	Source             string   `json:"source"`
	License            []string `json:"license"`
	Version            string   `json:"version"`
	Os                 string   `json:"os"`
	Architecture       string   `json:"architecture"`
	Maintainer         string   `json:"maintainer"`
	OriginalMaintainer string   `json:"originalMaintainer"`
	Homepage           string   `json:"homepage"`
	Description        string   `json:"description"`
	Depends            string   `json:"depends"`
	Hashes             []Hash   `json:"hashes"`
}

type Hash struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
