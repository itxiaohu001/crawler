package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestApk_Parse(t *testing.T) {
	apk := NewApkParser()
	fileList := []string{
		"E:\\tmp\\test\\acf-provisioning-snom-8.4.32-r0.apk",
		"E:\\tmp\\test\\freeswitch-sounds-music-32000-1.0.8-r1.apk",
		"E:\\tmp\\test\\g++-4.8.2-r10.apk",
	}
	for i, path := range fileList {
		f, err := os.Open(path)
		if err != nil {
			t.Error(err)
			continue
		}
		if err := apk.Parse(f, filepath.Join(filepath.Dir(path), fmt.Sprintf("%d.json", i))); err != nil {
			t.Error(err)
			continue
		}
		f.Close()
	}
}

func TestDeb_Parse(t *testing.T) {
	deb := NewDebParser()
	fileList := []string{
		//"E:\\tmp\\test\\jabber-querybot_0.1.0-1.1_all.deb",
		//"E:\\tmp\\test\\jarwrapper_0.72.1~18.04.1_all.deb",
		//"E:\\tmp\\test\\libjboss-vfs-java_3.2.15.Final-2_all.deb",
		"E:\\tmp\\test\\update-manager_22.04.9_all.deb",
	}
	for i, path := range fileList {
		f, err := os.Open(path)
		if err != nil {
			t.Error(err)
			continue
		}
		if err := deb.Parse(f, filepath.Join(filepath.Dir(path), fmt.Sprintf("%d.json", i))); err != nil {
			t.Error(err)
			continue
		}
		f.Close()
	}
}
