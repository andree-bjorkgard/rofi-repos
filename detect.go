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
)

func DetectLanguage(folder string) string {
	files, err := ioutil.ReadDir(folder)
	if err == nil {
		prio := ^Priority(0)
		lang := ""
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

			case ".json":
				if prio <= PrioTypescript {
					continue
				}

				if name == "tsconfig.json" {
					prio = PrioTypescript
					lang = LanguageTypescript
					continue
				} else if name == "package.json" && prio <= PrioJavascript {
					prio = PrioJavascript
					lang = LanguageJavascript
					continue
				}

			case ".js":
				if prio <= PrioJavascript {
					continue
				}
				prio = PrioJavascript
				lang = LanguageJavascript

			case ".sh":
				if prio <= PrioShell {
					continue
				}
				prio = PrioShell
				lang = LanguageShell

			case ".properties":
				if prio <= PrioJava {
					continue
				}

				if name == "application.properties" || name == "system.properties" {
					prio = PrioJava
					lang = LanguageJava
				}

			}

		}
		return lang
	}

	return ""
}
