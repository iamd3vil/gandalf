package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
)

func main() {
	pkg := flag.String("pkg", "main", "Package to be given for the generated code")
	dir := flag.String("dir", ".", `Path of the directory for finding source files with structs.`)
	dest := flag.String("file", "", "Destination File")
	flag.Parse()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, *dir, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Couldn't parse directory: %v", err)
	}
	sts := parseNode(pkgs[*pkg])

	fs, err := initFileSystem()
	if err != nil {
		fmt.Printf("error while reading templates: %v\n", err)
		os.Exit(1)
	}

	code := bytes.NewBuffer([]byte{})
	err = generateCodeForStructs(fs, *pkg, sts, code)
	if err != nil {
		log.Fatalf("error while generating code: %v", err)
	}

	fPath := *dest
	if fPath == "" {
		fPath = fmt.Sprintf("%s_validate_gen.go", *pkg)
	}

	f, err := os.Create(fPath)
	if err != nil {
		fmt.Printf("error while creating file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fmted, err := format.Source(code.Bytes())
	if err != nil {
		log.Fatalf("error while formatting code: %v", err)
	}

	wtr := bufio.NewWriter(f)
	defer wtr.Flush()
	if _, err := wtr.Write(fmted); err != nil {
		log.Fatalf("error while storing the file: %v", err)
	}

}
