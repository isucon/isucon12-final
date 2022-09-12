package data

import (
	"encoding/json"
	"fmt"
	"os"
)

func (s *Set[T]) LoadJSON(jsonFile string) error {
	file, err := os.Open(jsonFile)
	if err != nil {
		return err
	}
	defer file.Close()

	models := []T{}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&models); err != nil {
		return err
	}

	for _, model := range models {
		if !s.Add(model) {
			return fmt.Errorf("unexpected error on dump loading: %v", model)
		}
	}

	return nil
}
