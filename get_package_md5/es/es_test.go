package es

import (
	testing2 "testing"
)

func TestInsert(t *testing2.T) {
	//pkg1 := &model.DebPkg{
	//	Name:    "huyongfeng",
	//	Version: "huyongfeng",
	//	Hashes: map[string]string{
	//		"234567810": "usr/local/bin/test",
	//	},
	//	Depends: []string{"deps1", "deps2"},
	//}
	//pkg2 := &model.DebPkg{
	//	Name:    "huyongfeng2",
	//	Version: "huyongfeng2",
	//	Hashes: map[string]string{
	//		"2345678910": "usr/local/bin/test3",
	//	},
	//	Depends: []string{"deps1", "deps2"},
	//}
	//cps := []*model.CommonPkg{model.Convert(pkg1, "pkg_bin_hash"), model.Convert(pkg2, "pkg_bin_hash")}

	v2, _ := NewClientV2("http://127.0.0.1:9200")
	if _, err := v2.Search("pkg_bin_hash", "2345678910"); err != nil {
		t.Error(err)
		return
	}

	//err := v2.Insert("pkg_bin_hash", cp)
	//if err != nil {
	//	t.Error(err)
	//	return
	//}

	//err := v2.MultInsert("pkg_bin_hash", cps)
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
}
