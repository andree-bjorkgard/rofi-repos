# Flatten Sub-Projects Into Main Rofi List

## Problem

Monorepo sub-projects currently require a second keypress: selecting the monorepo opens a second rofi window showing sub-projects. This extra step is unnecessary friction.

## Solution

Display sub-projects as flat entries in the main rofi list with `repo/subproject` labels. Remove the two-step navigation entirely.

## Approach

Flatten at display time (Approach 1). No changes to the indexer or data model. `CategorizedRepo` keeps its `SubProjects` field. All changes are in `cmd/lister/rofi-repos.go`.

## Design

### Main list building (default mode)

When iterating repos, if a repo has `SubProjects`:

1. Add the parent repo as a normal entry (label: repo name, command: `editor-save`).
2. For each sub-project, add a flattened entry:
   - Label: `{parentName}/{subProjectName}` (last path component of sub-project).
   - If two sub-projects share the same name across base paths, use the full relative path.
   - Value: sub-project's full path.
   - Command: `editor-save` (same as regular repos).
   - Icon/category: sub-project's own detected language.

If a repo has no sub-projects, behavior is unchanged.

### Removals

- The `"subprojects"` command handler and its rofi mode.
- The `"subproject-editor-save"` command handler.
- The `"recent_subprojects"` history namespace.
- The logic that sets `Cmds: ["subprojects", "context-menu"]` for monorepos.

### History

All entries (parent repos and sub-projects) use the single `"recent_repos"` namespace. Selecting a sub-project saves it to history like any other repo.

### Name collision handling

Collect all sub-project names across all base paths for a given monorepo. If duplicates exist, fall back to the relative path for those entries.

## Scope

- Only `cmd/lister/rofi-repos.go` changes.
- No indexer changes.
- No data model changes.
