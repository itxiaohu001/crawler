package utils

import "os"

func RecordErrors(error error, path string) {
	f, _ := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	defer f.Close()
	f.WriteString(error.Error() + "\n")
}

func RecordDownloaded(p string, path string) {
	f, _ := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	defer f.Close()
	f.WriteString(p + "\n")
}
