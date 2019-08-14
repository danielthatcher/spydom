package main

import (
	"bufio"
	"encoding/base64"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/packr"
	"gitlab.com/dcthatch/spydom/config"
)

func report(conf *config.Config) error {
	// Load the report template
	box := packr.NewBox("./templates")
	t, err := template.New("report-main").Funcs(template.FuncMap{
		"join": func(paths ...string) string {
			return path.Join(paths...)
		},
		"embedFile": func(p string) string {
			b, err := ioutil.ReadFile(p)
			if err != nil {
				log.Printf("Failed to read file '%s' while generating template: %v\n", p, err)
			}
			s := string(b)
			s = strings.TrimRight(s, "\n")
			s = strings.TrimRight(s, "\r")
			return s
		},
		"embedPNG": func(p string) template.URL {
			b, err := ioutil.ReadFile(p)
			if err != nil {
				log.Printf("Failed to read file '%s' while generating template: %v\n", p, err)
			}
			encoded := base64.StdEncoding.EncodeToString(b)
			return template.URL("data:image/png;base64," + encoded)
		},
		"embedBeautified": func(dir string) template.HTML {
			// Glob for all beautified lsiteners
			g := path.Join(dir, "*.beautified")
			matches, err := filepath.Glob(g)
			if err != nil {
				log.Printf("Error finding beatufied JS files in %s: %v\n", dir, err)
			}

			// Construct HTML output
			var buf string
			for _, m := range matches {
				name := strings.Replace(filepath.Base(m), ".beautified", "", 1)
				buf += "// " + name + "\n"
				contents, err := ioutil.ReadFile(m)
				if err != nil {
					log.Printf("Failed to read message listener file %s: %v\n", m, err)
				}
				buf += string(contents)
				buf += "\n"
			}

			if buf == "" {
				return template.HTML("None")
			}
			return template.HTML(buf)
		},
	}).Parse(box.String("index.html"))
	if err != nil {
		log.Fatalf("Error compiling report template: %v\n", err)
	}

	// Load all the URLS to pass to the template
	urlsFile, err := os.Open(conf.URLsFile)
	defer urlsFile.Close()
	if err != nil {
		log.Fatalf("Failed to open URLs file when generating report: %v\n", err)
	}

	// Dirs holds all the directories for the output
	var dirs []string
	scanner := bufio.NewScanner(urlsFile)
	for scanner.Scan() {
		rel := getRelDir(scanner.Text())
		dirs = append(dirs, path.Join(conf.OutDir, rel))
	}

	// Get the output file
	outFile, err := os.Create(conf.ReportFile)
	defer outFile.Close()
	w := bufio.NewWriter(outFile)

	// Execute the template
	err = t.Execute(w, dirs)
	if err != nil {
		log.Fatalf("Failed to execute report template: %v\n", err)
	}
	w.Flush()

	return nil
}
