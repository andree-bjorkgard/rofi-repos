package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/ingentingalls/rofi"

	"github.com/ingentingalls/rofi-repos/pkg/config"
	"github.com/ingentingalls/rofi-repos/pkg/repo"
)

const namespace = "recent_repos"

// subProjectLabels returns display labels for sub-projects.
// Uses "parentName/subProjectName" unless names collide,
// in which case falls back to "parentName/relativePath".
func subProjectLabels(parentName, parentPath string, subProjects []repo.CategorizedRepo) map[string]string {
	nameCount := make(map[string]int)
	for _, sp := range subProjects {
		nameCount[path.Base(sp.Path)]++
	}

	labels := make(map[string]string, len(subProjects))
	for _, sp := range subProjects {
		baseName := path.Base(sp.Path)
		if nameCount[baseName] > 1 {
			rel := strings.TrimPrefix(sp.Path, parentPath+"/")
			labels[sp.Path] = parentName + "/" + rel
		} else {
			labels[sp.Path] = parentName + "/" + baseName
		}
	}

	return labels
}

func main() {
	rofi.EnableHotkeys()

	opts := rofi.Options{}

	if rofi.GetVerbosityLevel() > 2 {
		log.Println("State:", rofi.GetState())
	}

	val := rofi.GetValue()
	if val == nil {
		val = &rofi.Value{}
	}

	if rofi.GetVerbosityLevel() >= 2 {
		log.Printf("Cmd: %s, Value: %s\n", val.Cmd, val.Value)

		if rofi.GetVerbosityLevel() >= 5 {
			os.Exit(0)
		}
	}

	var cmd *exec.Cmd

	switch val.Cmd {
	case "terminal":
		cmd = exec.Command("i3-sensible-terminal", "--working-directory", val.Value)
	case "clipboard":
		cmd = exec.Command("xsel", "--input", "--clipboard")
		cmd.Stdin = strings.NewReader(val.Value)

		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}

		os.Exit(0)

	case "url":
		cmd = exec.Command("xdg-open", val.Value)

	case "editor-save":
		rofi.SaveToHistory(namespace, val.Value)
		fallthrough
	case "editor":
		cmd = exec.Command("i3-sensible-terminal", "--working-directory", val.Value, "-e", "i3-sensible-editor")

	case "context-menu":
		rofi.SaveToHistory(namespace, val.Value)
		rofi.SetPrompt("")
		rofi.SetMessage(path.Base(val.Value))

		opts = append(opts, rofi.Option{
			Label: "Open in editor",
			Icon:  "",
			Value: val.Value,
			Cmds:  []string{"editor"},
		},
			rofi.Option{
				Label: "Open in terminal",
				Icon:  "Terminal",
				Value: val.Value,
				Cmds:  []string{"terminal"},
			},
		)

		if _, err := exec.LookPath("git"); err == nil {
			cmd := exec.Command("git", "-C", val.Value, "config", "--get", "remote.origin.url")
			out, err := cmd.Output()

			if err == nil {
				url := string(out)
				url = strings.TrimSpace(url)
				if strings.Contains(url, "git@") {
					url = strings.TrimPrefix(url, "git@")
					url = strings.TrimSuffix(url, ".git")
					url = strings.ReplaceAll(url, ":", "/")
					url = "https://" + url
				}

				opts = append(opts, rofi.Option{
					Label: "Visit repository in browser",
					Icon:  "github",
					Value: url,
					Cmds:  []string{"url"},
				})
			}
		}

		if _, err := exec.LookPath("xsel"); err == nil {
			opts = append(opts, rofi.Option{
				Label: "Copy path to clipboard",
				Icon:  "gtk-copy",
				Value: val.Value,
				Cmds:  []string{"clipboard"},
			})
		}

		opts = append(opts, rofi.Option{
			Label: "Go back",
			Icon:  "back",
			Cmds:  []string{"back"},
		})

	default:
		rofi.SetPrompt("")
		rofi.SetMessage("")
		rofi.UseHistory(namespace)
		rofi.EnableMarkup()

		cfg := config.GetListConfig()
		repos := repo.GetCategorizedRepos(cfg.RepoCachePath)

		for _, r := range repos {
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
		opts.Sort()
	}

	if cmd != nil {
		cmd.Start()
		os.Exit(0)
	}

	opts.PrintAll()
}
