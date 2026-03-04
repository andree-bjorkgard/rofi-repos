# Flatten Sub-Projects Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Display monorepo sub-projects as flat entries in the main rofi list (`repo/subproject`), removing the two-step navigation.

**Architecture:** All changes in `cmd/lister/rofi-repos.go`. The indexer and data model are untouched. When building the main option list, repos with `SubProjects` emit both a parent entry and one flattened entry per sub-project. Dead code from the old two-step flow is removed.

**Tech Stack:** Go, rofi Go bindings (`github.com/ingentingalls/rofi`)

---

### Task 1: Add helper to build sub-project labels with collision detection

**Files:**
- Modify: `cmd/lister/rofi-repos.go` (add function after line 18, before `func main()`)

**Step 1: Write the `subProjectLabel` helper function**

Add this function to `cmd/lister/rofi-repos.go`, right before `func main()`:

```go
// subProjectLabel returns a display label for each sub-project.
// Uses short form "parentName/subProjectName" unless names collide,
// in which case it falls back to "parentName/relativePath".
func subProjectLabels(parentName, parentPath string, subProjects []repo.CategorizedRepo) map[string]string {
	// Count occurrences of each sub-project base name
	nameCount := make(map[string]int)
	for _, sp := range subProjects {
		nameCount[path.Base(sp.Path)]++
	}

	labels := make(map[string]string, len(subProjects))
	for _, sp := range subProjects {
		baseName := path.Base(sp.Path)
		if nameCount[baseName] > 1 {
			// Collision: use relative path from parent
			rel := strings.TrimPrefix(sp.Path, parentPath+"/")
			labels[sp.Path] = parentName + "/" + rel
		} else {
			labels[sp.Path] = parentName + "/" + baseName
		}
	}

	return labels
}
```

**Step 2: Verify it compiles**

Run: `go build ./cmd/lister/`
Expected: no errors

**Step 3: Commit**

```bash
git add cmd/lister/rofi-repos.go
git commit -m "feat: add subProjectLabels helper for flat display"
```

---

### Task 2: Flatten sub-projects into the main list

**Files:**
- Modify: `cmd/lister/rofi-repos.go` — the `default` case in the switch (lines 182–215)

**Step 1: Replace the default case loop**

Replace lines 191–213 (the `for _, repo := range repos` loop) with:

```go
		for _, r := range repos {
			// Always add the parent repo as a normal entry
			opt := rofi.Option{
				Label:    r.Name,
				Value:    r.Path,
				Category: r.Language,
				Cmds:     []string{"editor-save", "context-menu"},
			}

			if r.Language != "" {
				opt.Icon = fmt.Sprintf("language-%s", r.Language)
			}

			if opt.Category != "" {
				opt.Category = fmt.Sprintf("<span style=\"italic\" size=\"10pt\" >(%s)</span>", opt.Category)
			}

			opts = append(opts, opt)

			// Flatten sub-projects into the main list
			if len(r.SubProjects) > 0 {
				labels := subProjectLabels(r.Name, r.Path, r.SubProjects)

				for _, sp := range r.SubProjects {
					spOpt := rofi.Option{
						Label:    labels[sp.Path],
						Value:    sp.Path,
						Category: sp.Language,
						Cmds:     []string{"editor-save", "context-menu"},
					}

					if sp.Language != "" {
						spOpt.Icon = fmt.Sprintf("language-%s", sp.Language)
					}

					if spOpt.Category != "" {
						spOpt.Category = fmt.Sprintf("<span style=\"italic\" size=\"10pt\" >(%s)</span>", spOpt.Category)
					}

					opts = append(opts, spOpt)
				}
			}
		}
```

Note: the loop variable is renamed from `repo` to `r` to avoid shadowing the `repo` import.

**Step 2: Verify it compiles**

Run: `go build ./cmd/lister/`
Expected: no errors

**Step 3: Commit**

```bash
git add cmd/lister/rofi-repos.go
git commit -m "feat: flatten sub-projects into main rofi list"
```

---

### Task 3: Remove dead subprojects code

**Files:**
- Modify: `cmd/lister/rofi-repos.go`

**Step 1: Remove the `subprojectNamespace` constant**

Delete line 18:
```go
const subprojectNamespace = "recent_subprojects"
```

**Step 2: Remove the `"subproject-editor-save"` case**

Delete the entire case block (lines 60–73):
```go
	case "subproject-editor-save":
		rofi.SaveToHistory(subprojectNamespace, val.Value)
		// Save the parent monorepo path to main history so it rises in the list
		cfg := config.GetListConfig()
		repos := repo.GetCategorizedRepos(cfg.RepoCachePath)
		for _, r := range repos {
			for _, sp := range r.SubProjects {
				if sp.Path == val.Value {
					rofi.SaveToHistory(namespace, r.Path)
					break
				}
			}
		}
		cmd = exec.Command("i3-sensible-terminal", "--working-directory", val.Value, "-e", "i3-sensible-editor")
```

**Step 3: Remove the `"subprojects"` case**

Delete the entire case block (lines 137–181):
```go
	case "subprojects":
		rofi.SetPrompt("")
		rofi.UseHistory(subprojectNamespace)
		rofi.EnableMarkup()
		...
		opts.Sort()
```

**Step 4: Remove unused imports if any**

After the deletions, check if `config` and `repo` imports are still used in the remaining code. The `default` case still uses both, so they should still be needed. But verify.

**Step 5: Verify it compiles**

Run: `go build ./cmd/lister/`
Expected: no errors

**Step 6: Commit**

```bash
git add cmd/lister/rofi-repos.go
git commit -m "refactor: remove two-step subprojects navigation code"
```

---

### Task 4: Manual smoke test

**Step 1: Rebuild and install**

Run: `go build -o rofi-repos ./cmd/lister/`

**Step 2: Verify the binary runs**

Run: `./rofi-repos` (will error without rofi context, but should not panic)

**Step 3: Test with rofi if available**

If the user has rofi running, test that:
- Regular repos appear as before
- Monorepo sub-projects appear as `reponame/subproject` entries
- Selecting a sub-project opens the editor directly (no second window)
- History works for both parent repos and sub-projects

**Step 4: Commit final state if any fixes were needed**

---

### Task 5: Clean up design docs

**Step 1: Remove the design and plan docs**

```bash
rm -rf docs/plans/
git add -A docs/
git commit -m "chore: remove design and implementation plan docs"
```
