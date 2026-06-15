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
	"strings"
	"syscall"
	"time"

	"golang.org/x/exp/slices"

	"github.com/andree-bjorkgard/rofi-repos/pkg/config"
	"github.com/andree-bjorkgard/rofi-repos/pkg/repo"
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
		entry := repo.CategorizedRepo{
			Name: path.Base(r),
			Path: r,
		}

		if monorepo := findMonorepoConfig(r, cfg.Monorepos); monorepo != nil {
			log.Printf("Monorepo detected: %s, scanning sub-projects", r)
			entry.SubProjects = discoverSubProjects(r, monorepo.SubProjects)
			log.Printf("Found %d sub-projects in %s", len(entry.SubProjects), r)
		}

		categorizedRepos = append(categorizedRepos, entry)
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

func discoverSubProjects(monorepoPath string, rules []config.SubProjectRule) []repo.CategorizedRepo {
	var subProjects []repo.CategorizedRepo
	seen := make(map[string]bool)

	for _, rule := range rules {
		searchRoot := path.Join(monorepoPath, rule.BasePath)

		info, err := os.Stat(searchRoot)
		if err != nil || !info.IsDir() {
			log.Printf("Monorepo scan path does not exist: %s", searchRoot)
			continue
		}

		maxDepth := rule.MaxDepth
		if maxDepth == 0 {
			maxDepth = 1
		}

		baseDepth := strings.Count(searchRoot, string(os.PathSeparator))

		filepath.Walk(searchRoot, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if !info.IsDir() {
				return nil
			}

			// Skip the search root itself
			if p == searchRoot {
				return nil
			}

			currentDepth := strings.Count(p, string(os.PathSeparator)) - baseDepth

			if currentDepth > maxDepth {
				return filepath.SkipDir
			}

			// If marker is set, check if this directory contains the marker file
			if rule.Marker != "" {
				markerPath := path.Join(p, rule.Marker)
				if _, err := os.Stat(markerPath); err != nil {
					// No marker file here; continue walking deeper
					return nil
				}
			}

			if seen[p] {
				// Already matched by an earlier rule; skip descent to avoid re-walking
				return filepath.SkipDir
			}
			seen[p] = true

			subProjects = append(subProjects, repo.CategorizedRepo{
				Name: path.Base(p),
				Path: p,
			})

			// Don't descend into a matched sub-project
			return filepath.SkipDir
		})
	}

	return subProjects
}

func findMonorepoConfig(repoPath string, monorepos []config.MonorepoConfig) *config.MonorepoConfig {
	for i, m := range monorepos {
		if m.Path == repoPath {
			return &monorepos[i]
		}
	}
	return nil
}
