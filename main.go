package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/ingentingalls/rofi"
)

const namespace = "recent_repos"

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

	case "code-save":
		rofi.SaveToHistory(namespace, val.Value)
		fallthrough
	case "code":
		cmd = exec.Command("code", val.Value)

	case "context-menu":
		rofi.SaveToHistory(namespace, val.Value)
		rofi.SetPrompt("")
		rofi.SetMessage(path.Base(val.Value))

		opts = append(opts, rofi.Option{
			Label: "Open in VSCode",
			Icon:  "visual-studio-code",
			Value: val.Value,
			Cmds:  []string{"code"},
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
					Label: "Show on Github",
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

		stringPaths := os.Getenv("REPO_PATHS")
		if stringPaths == "" {
			log.Println("error: Env REPO_PATHS is empty")
			return
		}

		folders := strings.Split(stringPaths, ";")

		for _, folder := range folders {
			repos, err := ioutil.ReadDir(folder)
			if err != nil {
				continue
			}

			for _, repo := range repos {
				if !repo.IsDir() {
					// Is not a repo
					continue
				}

				opt := rofi.Option{
					Label: repo.Name(),
					Value: path.Join(folder, repo.Name()),
					Cmds:  []string{"code-save", "context-menu"},
				}

				opt.Category, opt.Icon = DetectLanguage(opt.Value)

				if opt.Category != "" {
					opt.Category = fmt.Sprintf("<span style=\"italic\" size=\"10pt\" >(%s)</span>", opt.Category)
				}

				opts = append(opts, opt)
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
