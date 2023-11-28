package utils

import (
	"encoding/json"
	"os"
)

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

func SaveJson(data interface{}, out string) error {
	f, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(data)
}
