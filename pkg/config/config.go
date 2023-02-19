package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"text/template"

	"github.com/BurntSushi/toml"
)

const namespace = "rofi-repos"
const repoCacheName = "categorizedRepos"

type IndexerConfig struct {
	BaseDirectory string
	Blacklist     []string

	Interval      int
	RepoCachePath string
	RunOnStart    bool

	DryRun bool

	// Skippable directories while analyzing the language
	SkippableDirs []string `toml:"SkippableDirsWhileAnalyzing"`
}

const configTemplate = `# The base directory to start indexing from
BaseDirectory = "{{index . "Home"}}"

# Interval in minutes between each indexing
Interval = 240

# Run on start
RunOnStart = true

# Blacklist is used to prevent certain folders from being included when indexed
# when looking for repos
# Since repo-indexer only indexes from the base directory for .git files,
# all blacklisted directories should be relative to it
# 
# Uncomment to use
Blacklist = [".config", ".cache", ".local", ".cargo", ".oh-my-zsh"]

# Skipping folders
SkippableDirsWhileAnalyzing = [".git", "node_modules", "vendor"]
`

func GetIndexerConfig() IndexerConfig {
	var cfg IndexerConfig
	repoCache, err := getPathToCache(namespace, repoCacheName)
	if err != nil {
		log.Fatalf("error while reading cache path: %s\n", err)
	}
	cfg.RepoCachePath = repoCache

	configPath, err := getPathToConfig(namespace, "indexer")
	if err != nil {
		log.Printf("error while reading config path, skipping loading config: %s\n", err)
		return cfg
	}

	_, err = os.Stat(configPath)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path.Dir(configPath), 0755); err != nil {
			log.Printf("Error while creating path: %s\n", err)
			return cfg
		}

		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalln("No home dir was found")
		}

		t, err := template.New("config").Parse(configTemplate)
		if err != nil {
			log.Fatalf("Error while creating config template: %s\n", err)
		}

		f, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("Error while creating config from template: %s\n", err)
		}
		defer f.Close()

		if err = t.Execute(f, map[string]string{"Home": home}); err != nil {
			log.Printf("Error while creating config from template: %s\n", err)

			// cleaning up
			if err := os.Remove(f.Name()); err != nil {
				log.Printf("Error while cleaning up broken config file: %s\n", err)
			}

			os.Exit(1)
		}
	} else if err != nil {
		log.Fatalf("error finding config: %s", err)
	}

	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		log.Printf("error while decoding config: %s\n", err)
	}

	cfg.RepoCachePath = repoCache

	return cfg
}

type ListConfig struct {
	RepoCachePath string
}

func GetListConfig() ListConfig {
	var cfg ListConfig
	repoCache, err := getPathToCache(namespace, repoCacheName)
	if err != nil {
		log.Fatalf("error while reading cache path: %s\n", err)
	}

	cfg.RepoCachePath = repoCache

	return cfg
}

func getPathToConfig(namespace string, name string) (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return path.Join(config, fmt.Sprintf("/%s/%s.toml", namespace, name)), nil
}

func getPathToCache(namespace string, name string) (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return path.Join(cache, fmt.Sprintf("/%s/%s.json", namespace, name)), nil
}
