package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/knadh/stuffbin"
)

var (
	buildDate, buildVersion string
)

// defaultFileMode is used for the generated file when the destination does
// not already exist.
const defaultFileMode = 0644

func main() {
	pkg := flag.String("pkg", "main", "Package to be given for the generated code")
	dir := flag.String("dir", ".", `Path of the directory for finding source files with structs.`)
	dest := flag.String("file", "", "Destination File")
	flag.Parse()

	fs, err := initFileSystem()
	if err != nil {
		log.Fatalf("error while reading templates: %v", err)
	}

	if err := run(fs, *pkg, *dir, *dest); err != nil {
		log.Fatal(err)
	}
}

// run performs the actual work of parsing the source directory, generating
// validator code for the requested package and writing it to the
// destination file. It contains no flag/logging concerns so that it can be
// exercised directly from tests.
func run(fs stuffbin.FileSystem, pkg, dir, dest string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("couldn't parse directory: %w", err)
	}

	pkgNode, ok := pkgs[pkg]
	if !ok {
		found := make([]string, 0, len(pkgs))
		for name := range pkgs {
			found = append(found, name)
		}
		sort.Strings(found)
		return fmt.Errorf("package %q not found in %q; packages found: %s", pkg, dir, strings.Join(found, ", "))
	}

	sts, err := parseNode(pkgNode)
	if err != nil {
		return fmt.Errorf("error while parsing structs: %w", err)
	}

	code := bytes.NewBuffer([]byte{})
	if err := generateCodeForStructs(fs, pkg, sts, code); err != nil {
		return fmt.Errorf("error while generating code: %w", err)
	}

	fPath := dest
	if fPath == "" {
		fPath = fmt.Sprintf("%s_validate_gen.go", pkg)
	}

	fmted, err := format.Source(code.Bytes())
	if err != nil {
		return fmt.Errorf("error while formatting code: %w", err)
	}

	if err := atomicWriteFile(fPath, fmted); err != nil {
		return fmt.Errorf("error while storing the file: %w", err)
	}

	return nil
}

// atomicWriteFile writes data to path atomically: it writes to a temporary
// file in the same directory as path and renames it into place only once
// the write succeeds, so that a failure never leaves a partially written or
// truncated destination file behind. The temporary file is always cleaned
// up on error. If path already exists, its file mode is preserved for the
// new file; otherwise defaultFileMode is used.
func atomicWriteFile(path string, data []byte) (err error) {
	destDir := filepath.Dir(path)

	mode := os.FileMode(defaultFileMode)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode()
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("couldn't stat destination file: %w", statErr)
	}

	tmp, err := ioutil.TempFile(destDir, ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("couldn't create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("couldn't write temp file: %w", err)
	}

	if err = tmp.Close(); err != nil {
		return fmt.Errorf("couldn't close temp file: %w", err)
	}

	if err = os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("couldn't set file mode: %w", err)
	}

	if err = os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("couldn't rename temp file into place: %w", err)
	}

	return nil
}
