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
const subprojectNamespace = "recent_subprojects"

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

	case "subprojects":
		rofi.SetPrompt("")
		rofi.UseHistory(subprojectNamespace)
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
				Cmds:     []string{"subproject-editor-save", "context-menu"},
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

	default:
		rofi.SetPrompt("")
		rofi.SetMessage("")
		rofi.UseHistory(namespace)
		rofi.EnableMarkup()

		cfg := config.GetListConfig()
		repos := repo.GetCategorizedRepos(cfg.RepoCachePath)

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

			if repo.Language != "" {
				opt.Icon = fmt.Sprintf("language-%s", repo.Language)
			}

			if opt.Category != "" {
				opt.Category = fmt.Sprintf("<span style=\"italic\" size=\"10pt\" >(%s)</span>", opt.Category)
			}

			opts = append(opts, opt)
		}
		opts.Sort()
	}

	if cmd != nil {
		cmd.Start()
		os.Exit(0)
	}

	opts.PrintAll()
}
