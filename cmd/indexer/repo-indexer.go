package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-enry/go-enry/v2"
	"golang.org/x/exp/slices"

	"github.com/ingentingalls/rofi-repos/pkg/config"
	"github.com/ingentingalls/rofi-repos/pkg/repo"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Dry run simulates indexing but prints to stdout instead of file")
	daemon := flag.Bool("daemon", false, "If it should start in daemon mode and index on an interval. Otherwise it just indexes once and then quits")
	flag.Parse()

	cfg := config.GetIndexerConfig()
	cfg.DryRun = *dryRun

	if *daemon {
		log.Println("Starting daemon")

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		if cfg.RunOnStart {
			log.Println("Trigger index on startup")
			index(cfg)
		}

		for {
			log.Println("Sleeping")
			select {
			case <-c:
				log.Println("Got interrupt signal. Shutting down.")
				os.Exit(0)

			case <-time.After(time.Minute * time.Duration(cfg.Interval)):
				index(cfg)
			}
		}

	} else {
		index(cfg)
	}
}

func index(cfg config.IndexerConfig) {
	log.Println("Indexing")
	var repos []string

	filepath.Walk(cfg.BaseDirectory, func(p string, info os.FileInfo, err error) error {
		if slices.IndexFunc(cfg.Blacklist, func(blacklisted string) bool { return path.Join(cfg.BaseDirectory, blacklisted) == p }) != -1 {
			return filepath.SkipDir
		}

		if errors.Is(err, fs.ErrPermission) {
			return nil
		} else if err != nil {
			log.Fatalln(err)
		}

		if info.Name() == ".git" {
			repos = append(repos, path.Dir(p))
			return filepath.SkipDir
		}

		return nil
	})

	var categorizedRepos []repo.CategorizedRepo

	for _, r := range repos {
		langProbability := map[string]int{}

		filepath.Walk(r, func(p string, info os.FileInfo, err error) error {
			if slices.IndexFunc(cfg.SkippableDirs, func(dir string) bool { return dir == p }) != -1 {
				return filepath.SkipDir
			}

			if errors.Is(err, fs.ErrPermission) {
				return nil
			} else if err != nil {
				log.Printf("Could not traverse file (%s): %s", info.Name(), err)
				return nil
			}

			if info.IsDir() {
				return nil
			}

			lang, safe := enry.GetLanguageByExtension(p)
			if lang != "" && safe {
				langProbability[lang]++
			}

			return nil
		})

		// detect language that the repo uses
		topLang := ""
		for lang, probability := range langProbability {
			if topLang == "" || probability > langProbability[topLang] {
				topLang = lang
			}
		}

		categorizedRepos = append(categorizedRepos, repo.CategorizedRepo{
			Name:     path.Base(r),
			Path:     r,
			Language: topLang,
		})
	}

	if !cfg.DryRun {
		if err := repo.SaveCategorizedRepos(cfg.RepoCachePath, categorizedRepos); err != nil {
			log.Fatalf("Failed while saving categorized repos: %s", err)
		}
	} else {
		output, err := json.MarshalIndent(categorizedRepos, "", "  ")
		if err != nil {
			log.Println("Failed while trying to marshall output during dry-run")
		}

		fmt.Println(string(output))
	}

	log.Println("Indexing complete")
}
