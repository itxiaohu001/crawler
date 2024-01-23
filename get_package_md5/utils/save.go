package utils

import (
	"encoding/json"
	"os"
)

func SaveJson(data interface{}, out string) error {
	f, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(data)
}
