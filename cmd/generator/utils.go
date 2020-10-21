package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"

	"github.com/knadh/stuffbin"
)

// initFileSystem initializes the stuffbin FileSystem to provide
// access to bunded static assets to the app.
func initFileSystem() (stuffbin.FileSystem, error) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	exPath := filepath.Dir(ex)
	fs, err := stuffbin.UnStuff(filepath.Join(exPath, filepath.Base(os.Args[0])))
	if err != nil {
		return nil, err
	}

	return fs, nil
}

// parse takes in a template path and the variables to be "applied" on it. The rendered template
// is saved to the destination path.
func parse(name string, templateNames []string, fs stuffbin.FileSystem) (*template.Template, error) {
	tmpl := template.New(name)

	for _, t := range templateNames {
		// read template file
		c, err := fs.Read(t)
		if err != nil {
			return nil, fmt.Errorf("error reading template: %v", err)
		}

		tmpl, err = tmpl.Parse(string(c))
		if err != nil {
			return nil, fmt.Errorf("error parsing template: %v", err)
		}
	}

	return tmpl, nil
}

func writeTemplate(tmpl *template.Template, config map[string]interface{}, dest io.Writer) error {
	// apply the variable and save the rendered template to dest.
	return tmpl.Execute(dest, config)
}

func saveResource(name string, templateNames []string, dest io.Writer, config map[string]interface{}, fs stuffbin.FileSystem) error {
	// parse template file
	tmpl, err := parse(name, templateNames, fs)
	if err != nil {
		return err
	}

	return writeTemplate(tmpl, config, dest)
}
