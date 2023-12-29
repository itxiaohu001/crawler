package model

type RpmPkg struct {
	Epoch              int      `json:"epoch,omitempty"`
	OS                 string   `json:"os,omitempty"`
	Vendor             string   `json:"vendor,omitempty"`
	Release            string   `json:"release,omitempty"`
	Manager            string   `json:"manager,omitempty"`
	Name               string   `json:"name,omitempty"`
	Source             string   `json:"source,omitempty"`
	Version            string   `json:"version,omitempty"`
	Architecture       string   `json:"architecture,omitempty"`
	Maintainer         string   `json:"maintainer,omitempty"`
	OriginalMaintainer string   `json:"originalMaintainer,omitempty"`
	Homepage           string   `json:"homepage,omitempty"`
	Description        string   `json:"description,omitempty"`
	Depends            []string `json:"depends,omitempty"`
	License            []string `json:"license,omitempty"`
	Hashes             []Hash   `json:"hashes,omitempty"`
}

type DebPkg struct {
	OS                 string            `json:"os,omitempty"`
	OriginName         string            `json:"originName,omitempty"`
	Name               string            `json:"name,omitempty"`
	Source             string            `json:"source,omitempty"`
	Version            string            `json:"version,omitempty"`
	Architecture       string            `json:"architecture,omitempty"`
	Maintainer         string            `json:"maintainer,omitempty"`
	OriginalMaintainer string            `json:"originalMaintainer,omitempty"`
	Section            string            `json:"section,omitempty"`
	Priority           string            `json:"priority,omitempty"`
	Homepage           string            `json:"homepage,omitempty"`
	Description        string            `json:"description,omitempty"`
	InstalledSize      string            `json:"installedSize,omitempty"`
	Suggests           []string          `json:"suggests,omitempty"`
	Depends            []string          `json:"depends,omitempty"`
	Licences           []*License        `json:"licence,omitempty"`
	Hashes             map[string]string `json:"hashes,omitempty"`
}

type ApkPkg struct {
	OS         string   `json:"OS,omitempty"`
	PkgName    string   `json:"pkgname,omitempty"`
	PkgVer     string   `json:"pkgver,omitempty"`
	PkgDesc    string   `json:"pkgdesc,omitempty"`
	URL        string   `json:"url,omitempty"`
	BuildDate  string   `json:"builddate,omitempty"`
	Packager   string   `json:"packager,omitempty"`
	Size       string   `json:"size,omitempty"`
	Arch       string   `json:"arch,omitempty"`
	Origin     string   `json:"origin,omitempty"`
	Maintainer string   `json:"maintainer,omitempty"`
	Replaces   string   `json:"replaces,omitempty"`
	Depend     []string `json:"depend,omitempty"`
	License    []string `json:"license,omitempty"`
	Hashes     []Hash   `json:"hashes,omitempty"`
}

type License struct {
	Names []string `json:"names"`
	Per   float64  `json:"per"`
	Path  string   `json:"path"`
}

type CommonPkg struct {
	Manager            string   `json:"manager"`
	Name               string   `json:"name"`
	Source             string   `json:"source"`
	Version            string   `json:"version"`
	Architecture       string   `json:"architecture"`
	Maintainer         string   `json:"maintainer"`
	OriginalMaintainer string   `json:"originalMaintainer"`
	Homepage           string   `json:"homepage"`
	Description        string   `json:"description"`
	Depends            []string `json:"depends"`
	License            []string `json:"license"`
	Hashes             []Hash   `json:"hashes"`
}

func Convert(pkg *DebPkg, mt string) *CommonPkg {
	cp := new(CommonPkg)
	if pkg == nil {
		return cp
	}
	cp.Name = pkg.Name
	cp.Source = pkg.Source
	cp.Version = pkg.Version
	cp.Architecture = pkg.Architecture
	cp.Manager = mt
	cp.Homepage = pkg.Homepage
	cp.Maintainer = pkg.Maintainer
	cp.OriginalMaintainer = pkg.OriginalMaintainer
	cp.Depends = pkg.Depends
	cp.Description = pkg.Description
	for k, v := range pkg.Hashes {
		cp.Hashes = append(cp.Hashes, Hash{
			k,
			v,
		})
	}
	for _, l := range pkg.Licences {
		if l != nil {
			cp.License = append(cp.License, RemoveDuplicates(l.Names)...)
		}
	}
	return cp
}

func RemoveDuplicates(slice []string) []string {
	encountered := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !encountered[item] {
			encountered[item] = true
			result = append(result, item)
		}
	}

	return result
}

func (p *DebPkg) Merge(n *DebPkg) {
	if p == nil || n == nil {
		return
	}
	if n.Name != "" {
		p.Name = n.Name
	}
	if n.Version != "" {
		p.Version = n.Version
	}
	if n.Architecture != "" {
		p.Architecture = n.Architecture
	}
	if len(n.Depends) != 0 {
		p.Depends = n.Depends
	}
	if n.Maintainer != "" {
		p.Maintainer = n.Maintainer
	}
	if n.OriginalMaintainer != "" {
		p.OriginalMaintainer = n.OriginalMaintainer
	}
	if n.Section != "" {
		p.Section = n.Section
	}
	if n.Priority != "" {
		p.Priority = n.Priority
	}
	if n.Homepage != "" {
		p.Homepage = n.Homepage
	}
	if n.Description != "" {
		p.Description = n.Description
	}
	if n.InstalledSize != "" {
		p.InstalledSize = n.InstalledSize
	}
	if len(n.Suggests) != 0 {
		p.Suggests = n.Suggests
	}
	if len(n.Hashes) != 0 {
		p.Hashes = n.Hashes
	}
	if len(n.Licences) != 0 {
		p.Licences = n.Licences
	}
}
