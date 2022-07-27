package main

import (
	"io/ioutil"
	"path"
)

type Priority uint8

const (
	PrioGolang Priority = 0
	PrioTypescript
	PrioJava       Priority = 2
	PrioJavascript Priority = 3
	PrioShell      Priority = 5
)

const (
	LanguageGolang     = "Golang"
	LanguageTypescript = "Typescript"
	LanguageJavascript = "Javascript"
	LanguageJava       = "Java"
	LanguageShell      = "Shell"

	IconGolang     = "language-golang"
	IconTypescript = "language-typescript"
	IconJavascript = "language-javascript"
	IconJava       = "language-java"
	IconShell      = "Terminal"
)

func DetectLanguage(folder string) (string, string) {
	lang, icon := "", ""

	files, err := ioutil.ReadDir(folder)
	if err == nil {
		prio := ^Priority(0)
		for _, file := range files {
			if prio == 0 {
				break
			}

			if file.IsDir() {
				continue
			}
			ext := path.Ext(file.Name())
			name := file.Name()

			switch ext {
			case ".mod", ".go":
				if prio <= PrioGolang {
					continue
				}

				prio = PrioGolang
				lang = LanguageGolang
				icon = IconGolang

			case ".json":
				if prio <= PrioTypescript {
					continue
				}

				if name == "tsconfig.json" {
					prio = PrioTypescript
					lang = LanguageTypescript
					icon = IconTypescript
					continue
				} else if name == "package.json" && prio <= PrioJavascript {
					prio = PrioJavascript
					lang = LanguageJavascript
					icon = IconJavascript
					continue
				}

			case ".js":
				if prio <= PrioJavascript {
					continue
				}
				prio = PrioJavascript
				lang = LanguageJavascript
				icon = IconJavascript

			case ".sh":
				if prio <= PrioShell {
					continue
				}
				prio = PrioShell
				lang = LanguageShell
				icon = IconShell

			case ".properties":
				if prio <= PrioJava {
					continue
				}

				if name == "application.properties" || name == "system.properties" {
					prio = PrioJava
					lang = LanguageJava
					icon = IconJava
				}

			}

		}
	}

	return lang, icon
}
