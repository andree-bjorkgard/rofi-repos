package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
)

type CategorizedRepo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Language string `json:"language"`
}

func GetCategorizedRepos(cachePath string) []CategorizedRepo {
	var repos []CategorizedRepo

	b, err := os.ReadFile(cachePath)
	if err != nil {
		log.Fatalf("Error while reading the repo cache file: %s", err)
	}

	if err = json.Unmarshal(b, &repos); err != nil {
		log.Fatalf("Error while unmarshalling the repo cache file: %s", err)
	}

	return repos
}

func SaveCategorizedRepos(cachePath string, repos []CategorizedRepo) error {
	_, err := os.Stat(cachePath)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path.Dir(cachePath), 0755); err != nil {
			return fmt.Errorf("error while creating path: %w", err)
		}
	}

	b, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return fmt.Errorf("error while reading the repo cache file: %w", err)
	}

	if err := os.WriteFile(cachePath, b, 0644); err != nil {
		return fmt.Errorf("error while unmarshalling the repo cache file: %w", err)
	}

	return nil
}
