# Monorepo Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add configurable monorepo sub-project discovery to the indexer and two-step Rofi navigation to the lister.

**Architecture:** The indexer gains a monorepo scanning phase that runs after `.git` discovery. When a discovered repo matches a configured `Monorepos[].Path`, it applies `SubProjects` rules (BasePath + MaxDepth + optional Marker) to find sub-projects. The lister adds a `"subprojects"` command that displays a second Rofi menu for drilling into sub-projects.

**Tech Stack:** Go 1.20, BurntSushi/toml, go-enry/v2, ingentingalls/rofi

---

### Task 1: Add SubProjects field to CategorizedRepo

**Files:**
- Modify: `pkg/repo/repo.go:12-16`

**Step 1: Add the field**

In `pkg/repo/repo.go`, change the `CategorizedRepo` struct from:

```go
type CategorizedRepo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Language string `json:"language"`
}
```

to:

```go
type CategorizedRepo struct {
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Language    string            `json:"language"`
	SubProjects []CategorizedRepo `json:"subProjects,omitempty"`
}
```

**Step 2: Verify it compiles**

Run: `cd /home/andree/go/src/github.com/andree-bjorkgard/rofi-repos && go build ./...`
Expected: No errors. Existing code is unaffected because `SubProjects` defaults to nil.

**Step 3: Commit**

```bash
git add pkg/repo/repo.go
git commit -m "feat: add SubProjects field to CategorizedRepo"
```

---

### Task 2: Add monorepo config types

**Files:**
- Modify: `pkg/config/config.go:17-29`

**Step 1: Add the new types and field**

In `pkg/config/config.go`, add the following types before `IndexerConfig`:

```go
type SubProjectRule struct {
	BasePath string
	MaxDepth int
	Marker   string
}

type MonorepoConfig struct {
	Path        string
	SubProjects []SubProjectRule
}
```

Then add a `Monorepos` field to `IndexerConfig`:

```go
type IndexerConfig struct {
	BaseDirectory string
	Blacklist     []string

	Interval      int
	RepoCachePath string
	RunOnStart    bool

	DryRun bool

	// Skippable directories while analyzing the language
	SkippableDirs []string `toml:"SkippableDirsWhileAnalyzing"`

	Monorepos []MonorepoConfig
}
```

**Step 2: Verify it compiles**

Run: `cd /home/andree/go/src/github.com/andree-bjorkgard/rofi-repos && go build ./...`
Expected: No errors. TOML decoding handles new fields gracefully (missing = zero value).

**Step 3: Commit**

```bash
git add pkg/config/config.go
git commit -m "feat: add MonorepoConfig and SubProjectRule config types"
```

---

### Task 3: Extract detectLanguage helper in indexer

**Files:**
- Modify: `cmd/indexer/repo-indexer.go:85-125`

The current `index()` function has inline language detection logic (lines 85-125). Extract it into a reusable `detectLanguage(dir string, skippableDirs []string) string` function so it can be used for both regular repos and monorepo sub-projects.

**Step 1: Add the helper function**

Add this function after the `index()` function in `cmd/indexer/repo-indexer.go`:

```go
func detectLanguage(dir string, skippableDirs []string) string {
	langProbability := map[string]int{}

	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if slices.IndexFunc(skippableDirs, func(d string) bool { return d == p }) != -1 {
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

	topLang := ""
	for lang, probability := range langProbability {
		if topLang == "" || probability > langProbability[topLang] {
			topLang = lang
		}
	}

	return topLang
}
```

**Step 2: Replace inline logic with helper call**

In the `index()` function, replace the loop body (lines 85-125) from:

```go
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
```

to:

```go
	for _, r := range repos {
		categorizedRepos = append(categorizedRepos, repo.CategorizedRepo{
			Name:     path.Base(r),
			Path:     r,
			Language: detectLanguage(r, cfg.SkippableDirs),
		})
	}
```

**Step 3: Verify it compiles**

Run: `cd /home/andree/go/src/github.com/andree-bjorkgard/rofi-repos && go build ./...`
Expected: No errors.

**Step 4: Commit**

```bash
git add cmd/indexer/repo-indexer.go
git commit -m "refactor: extract detectLanguage helper for reuse"
```

---

### Task 4: Add sub-project discovery function

**Files:**
- Modify: `cmd/indexer/repo-indexer.go`

**Step 1: Write the discoverSubProjects function**

Add this function to `cmd/indexer/repo-indexer.go`:

```go
func discoverSubProjects(monorepoPath string, rules []config.SubProjectRule, skippableDirs []string) []repo.CategorizedRepo {
	var subProjects []repo.CategorizedRepo

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

			subProjects = append(subProjects, repo.CategorizedRepo{
				Name:     path.Base(p),
				Path:     p,
				Language: detectLanguage(p, skippableDirs),
			})

			// Don't descend into a matched sub-project
			return filepath.SkipDir
		})
	}

	return subProjects
}
```

Note: This requires adding `"strings"` to the import block. Check the existing imports first — if it's not there, add it.

**Step 2: Verify it compiles**

Run: `cd /home/andree/go/src/github.com/andree-bjorkgard/rofi-repos && go build ./...`
Expected: No errors.

**Step 3: Commit**

```bash
git add cmd/indexer/repo-indexer.go
git commit -m "feat: add discoverSubProjects function for monorepo scanning"
```

---

### Task 5: Wire monorepo scanning into index()

**Files:**
- Modify: `cmd/indexer/repo-indexer.go` — the `index()` function

**Step 1: Add monorepo lookup helper**

Add a helper function to find a monorepo config by path:

```go
func findMonorepoConfig(repoPath string, monorepos []config.MonorepoConfig) *config.MonorepoConfig {
	for i, m := range monorepos {
		if m.Path == repoPath {
			return &monorepos[i]
		}
	}
	return nil
}
```

**Step 2: Update the repo categorization loop**

Replace the repo categorization loop in `index()` from:

```go
	for _, r := range repos {
		categorizedRepos = append(categorizedRepos, repo.CategorizedRepo{
			Name:     path.Base(r),
			Path:     r,
			Language: detectLanguage(r, cfg.SkippableDirs),
		})
	}
```

to:

```go
	for _, r := range repos {
		entry := repo.CategorizedRepo{
			Name:     path.Base(r),
			Path:     r,
			Language: detectLanguage(r, cfg.SkippableDirs),
		}

		if monorepo := findMonorepoConfig(r, cfg.Monorepos); monorepo != nil {
			log.Printf("Monorepo detected: %s, scanning sub-projects", r)
			entry.SubProjects = discoverSubProjects(r, monorepo.SubProjects, cfg.SkippableDirs)
			log.Printf("Found %d sub-projects in %s", len(entry.SubProjects), r)
		}

		categorizedRepos = append(categorizedRepos, entry)
	}
```

**Step 3: Verify it compiles**

Run: `cd /home/andree/go/src/github.com/andree-bjorkgard/rofi-repos && go build ./...`
Expected: No errors.

**Step 4: Manual test with dry-run**

Add the GoMecenat monorepo config to `~/.config/rofi-repos/indexer.toml`:

```toml
[[Monorepos]]
Path = "/home/andree/go/src/mecenat.com/GoMecenat"

[[Monorepos.SubProjects]]
BasePath = "mecenat/src/services"
MaxDepth = 2
Marker = "main.go"

[[Monorepos.SubProjects]]
BasePath = "mecenat/src/mecenat"
MaxDepth = 1

[[Monorepos.SubProjects]]
BasePath = "mecenat/src/scripts"
MaxDepth = 1
Marker = "main.go"

[[Monorepos.SubProjects]]
BasePath = "mecenat/src/experimental"
MaxDepth = 1

[[Monorepos.SubProjects]]
BasePath = "mecenat/src/mecenat-js"
MaxDepth = 1
```

Then run:

```bash
cd /home/andree/go/src/github.com/andree-bjorkgard/rofi-repos && go run ./cmd/indexer/ -dry-run 2>/dev/null | jq '.[] | select(.subProjects != null) | {name, subProjectCount: (.subProjects | length)}'
```

Expected: GoMecenat entry with sub-projects populated. Verify count is roughly 87.

**Step 5: Commit**

```bash
git add cmd/indexer/repo-indexer.go
git commit -m "feat: wire monorepo sub-project scanning into index()"
```

---

### Task 6: Add "subprojects" command to lister

**Files:**
- Modify: `cmd/lister/rofi-repos.go:43-150`

**Step 1: Add encoding/json import**

Add `"encoding/json"` to the import block in `cmd/lister/rofi-repos.go` (needed to decode the sub-projects from the cache). Actually, check first — the lister loads repos via `repo.GetCategorizedRepos()` which returns `[]CategorizedRepo` already unmarshalled. So no extra import needed. The sub-projects are already in the struct.

**Step 2: Add the "subprojects" case**

In the `switch val.Cmd` block in `cmd/lister/rofi-repos.go`, add a new case before the `default` case:

```go
	case "subprojects":
		rofi.SetPrompt("")
		rofi.EnableMarkup()

		cfg := config.GetListConfig()
		repos := repo.GetCategorizedRepos(cfg.RepoCachePath)

		// Find the monorepo matching the selected value
		var subProjects []repo.CategorizedRepo
		for _, r := range repos {
			if r.Path == val.Value {
				subProjects = r.SubProjects
				rofi.SetMessage(r.Name)
				break
			}
		}

		for _, sp := range subProjects {
			opt := rofi.Option{
				Label:    sp.Name,
				Value:    sp.Path,
				Category: sp.Language,
				Cmds:     []string{"editor-save", "context-menu"},
			}

			if sp.Language != "" {
				opt.Icon = fmt.Sprintf("language-%s", sp.Language)
			}

			if opt.Category != "" {
				opt.Category = fmt.Sprintf("<span style=\"italic\" size=\"10pt\" >(%s)</span>", opt.Category)
			}

			opts = append(opts, opt)
		}

		opts = append(opts, rofi.Option{
			Label: "Go back",
			Icon:  "back",
			Cmds:  []string{"back"},
		})

		opts.Sort()
```

**Step 3: Update the default case for monorepos**

In the `default` case, change the `Cmds` for repos that have sub-projects. Replace:

```go
		for _, repo := range repos {
			opt := rofi.Option{
				Label:    repo.Name,
				Value:    repo.Path,
				Category: repo.Language,
				Cmds:     []string{"editor-save", "context-menu"},
			}
```

with:

```go
		for _, repo := range repos {
			cmds := []string{"editor-save", "context-menu"}
			if len(repo.SubProjects) > 0 {
				cmds = []string{"subprojects", "context-menu"}
			}

			opt := rofi.Option{
				Label:    repo.Name,
				Value:    repo.Path,
				Category: repo.Language,
				Cmds:     cmds,
			}
```

**Step 4: Verify it compiles**

Run: `cd /home/andree/go/src/github.com/andree-bjorkgard/rofi-repos && go build ./...`
Expected: No errors.

**Step 5: Commit**

```bash
git add cmd/lister/rofi-repos.go
git commit -m "feat: add subprojects command for monorepo two-step navigation"
```

---

### Task 7: Build, install, and end-to-end test

**Step 1: Run the indexer to populate the cache**

```bash
cd /home/andree/go/src/github.com/andree-bjorkgard/rofi-repos && go run ./cmd/indexer/
```

Expected: Indexing completes. Check logs for "Monorepo detected" and "Found N sub-projects" messages.

**Step 2: Verify cache content**

```bash
cat ~/.cache/rofi-repos/categorizedRepos.json | jq '.[] | select(.subProjects != null) | {name, count: (.subProjects | length)}'
```

Expected: GoMecenat with ~87 sub-projects.

**Step 3: Install and test with Rofi**

```bash
cd /home/andree/go/src/github.com/andree-bjorkgard/rofi-repos && go install ./cmd/...
```

Then launch `rofi-repos` from Rofi. Verify:
- GoMecenat appears in the main list
- Pressing Enter on GoMecenat shows sub-project list
- Sub-projects show language categories
- Selecting a sub-project opens editor at that path
- Context menu on GoMecenat works (opens at monorepo root)
- "Go back" returns to the main list

**Step 4: Commit any fixes if needed**
