package qurery

import (
	"strings"

	"github.com/pkg/errors"
)

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

type OS struct {
	Family string
	Name   string
}

func (o *OS) string() string {
	return o.Family + " " + o.Name
}

type PkgKeyMessage struct {
	OS      OS
	PkgName string
	Version string
	repos   []string
	nvrs    []string
	cpes    []string
}

func NewPkgKeyMessage(os, pkgName, Version string) (*PkgKeyMessage, error) {
	osnv := strings.Split(os, " ")
	if len(osnv) < 2 {
		return nil, errors.New("invalid os " + os)
	}
	if pkgName == "" {
		return nil, errors.New("none pkg name")
	}
	if Version == "" {
		return nil, errors.New("none version")
	}

	osnv[0] = strings.TrimSpace(osnv[0])
	osnv[1] = strings.TrimSpace(osnv[1])

	var repos []string
	if osnv[0] == "centos" || osnv[0] == "redhat" {
		repos = defaultContentSets[osnv[1]]
	}

	return &PkgKeyMessage{
		OS:      OS{Family: osnv[0], Name: osnv[1]},
		PkgName: pkgName,
		Version: Version,
		repos:   repos,
	}, nil
}

type VulnComponent struct {
	ComponentOwner   string        `json:"component_owner"`
	ComponentName    string        `json:"component_name"`
	Language         string        `json:"language"`
	Type             string        `json:"type"`
	XmirrorId        string        `json:"xmirror_id"`
	VulVersionRange  string        `json:"vul_version_range"`
	XmUpdateTime     string        `json:"xm_update_time"`
	VulVersionDetail []interface{} `json:"vul_version_detail"`
}

type cpe struct {
	Id string `json:"id"`
}
