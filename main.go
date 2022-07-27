package main

import (
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

	if val := rofi.GetValue(); val != nil {
		if rofi.GetVerbosityLevel() >= 2 {
			log.Printf("Modifier: %d, Cmd: %s, Value: %s\n", val.Modifier, val.Cmd, val.Value)

			if rofi.GetVerbosityLevel() >= 5 {
				os.Exit(0)
			}
		}

		// Only trigger on first selection
		if val.Cmd == "" {
			rofi.SaveToHistory(namespace, val.Value)
		}

		if val.Cmd == "" && val.Modifier == 0 {
			val.Cmd = "code"
		}

		if val.Cmd != "" {
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
			default:
				cmd = exec.Command(val.Cmd, val.Value)
			}
			cmd.Start()
			os.Exit(0)
		}

		rofi.SetPrompt("")
		rofi.SetMessage(path.Base(val.Value))

		opts = append(opts, rofi.Option{
			Name:  "Open in VSCode",
			Icon:  "visual-studio-code",
			Value: val.Value,
			Cmd:   "code",
		},
			rofi.Option{
				Name:  "Open in terminal",
				Icon:  "Terminal",
				Value: val.Value,
				Cmd:   "terminal",
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
					Name:  "Show on Github",
					Icon:  "github",
					Value: url,
					Cmd:   "url",
				})
			}
		}

		if _, err := exec.LookPath("xsel"); err == nil {
			opts = append(opts, rofi.Option{
				Name:  "Copy path to clipboard",
				Icon:  "gtk-copy",
				Value: val.Value,
				Cmd:   "clipboard",
			})
		}

		opts = append(opts, rofi.Option{
			Name: "Go back",
			Icon: "back",
		})

	} else {
		rofi.SetPrompt("")
		rofi.SetMessage("")
		rofi.UseHistory(namespace)

		stringPaths := os.Getenv("REPO_PATHS")
		if stringPaths == "" {
			return
		}

		folders := strings.Split(stringPaths, ";")

		for _, folder := range folders {
			files, err := ioutil.ReadDir(folder)
			if err != nil {
				continue
			}

			for _, file := range files {
				if !file.IsDir() {
					continue
				}

				opt := rofi.Option{
					Name:  file.Name(),
					Value: path.Join(folder, file.Name()),
				}

				opt.Category, opt.Icon = DetectLanguage(opt.Value)

				opts = append(opts, opt)
			}
		}
		opts.Sort()

	}

	opts.PrintAll()
}
