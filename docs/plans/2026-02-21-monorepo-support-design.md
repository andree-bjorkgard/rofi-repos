# Monorepo Sub-Project Navigation

## Problem

rofi-repos discovers repositories by finding `.git` directories. Monorepos like GoMecenat contain a single `.git` but house ~87 distinct sub-projects (services, libraries, scripts, etc.). Currently rofi-repos shows one entry for the entire monorepo, forcing the user to navigate from the root manually.

## Solution

Add configurable monorepo support: the indexer discovers sub-projects within monorepos using path + depth + optional marker rules, and the lister provides two-step Rofi navigation to drill into sub-projects.

## Data Model

`CategorizedRepo` gains a `SubProjects` field:

```go
type CategorizedRepo struct {
    Name        string            `json:"name"`
    Path        string            `json:"path"`
    Language    string            `json:"language"`
    SubProjects []CategorizedRepo `json:"subProjects,omitempty"`
}
```

A monorepo entry in the cache:

```json
{
  "name": "GoMecenat",
  "path": "/home/andree/go/src/mecenat.com/GoMecenat",
  "language": "Go",
  "subProjects": [
    { "name": "allspark", "path": ".../services/allspark", "language": "Go" },
    { "name": "mhttp", "path": ".../mecenat/mhttp", "language": "Go" }
  ]
}
```

Regular repos are unchanged (nil/empty `SubProjects`).

## Config

New types added to `IndexerConfig`:

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

Example TOML:

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

- `BasePath`: relative path within the monorepo to scan
- `MaxDepth`: how many directory levels deep to scan (default 1)
- `Marker`: optional file that must exist in a directory for it to qualify as a sub-project (e.g., `main.go`)

## Indexer Changes

When the indexer finds a `.git` directory whose parent path matches a configured `Monorepos[].Path`:

1. Skip the normal single-entry treatment
2. For each `SubProjects` rule:
   - Walk `BasePath` up to `MaxDepth` levels
   - If `Marker` is set, only include directories containing that file
   - For each qualifying directory, run the existing language detection
   - Collect results as `SubProjects` on the parent `CategorizedRepo`
3. The parent monorepo still gets its own language detection (based on overall file composition)

## Lister / Rofi UI Changes

Two-step navigation for monorepos:

**Main list:** Monorepos and regular repos appear together. The difference is in the command binding:
- Regular repo: `Cmds = ["editor-save", "context-menu"]` (unchanged)
- Monorepo: `Cmds = ["subprojects", "context-menu"]`

Pressing Enter on a monorepo drills into its sub-projects. The context-menu hotkey opens the standard context menu (open editor at root, terminal, browser, copy path).

**Sub-project list:** Shows all sub-projects with language categories and a "Go back" option. Each sub-project behaves like a regular repo with `Cmds = ["editor-save", "context-menu"]`.

## Files Changed

| File | Change |
|------|--------|
| `pkg/config/config.go` | Add `MonorepoConfig`, `SubProjectRule` types; add `Monorepos` field to `IndexerConfig` |
| `pkg/repo/repo.go` | Add `SubProjects` field to `CategorizedRepo` |
| `cmd/indexer/repo-indexer.go` | Add monorepo detection and sub-project scanning logic |
| `cmd/lister/rofi-repos.go` | Add `"subprojects"` command handler; render sub-project list with back navigation |
