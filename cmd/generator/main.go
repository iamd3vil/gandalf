package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"log"
)

func main() {
	pkg := flag.String("pkg", "main", "Package to be given for the generated code")
	dir := flag.String("dir", ".", `Path of the directory for finding source files with structs.`)
	flag.Parse()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, *dir, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Couldn't parse directory: %v", err)
	}
	fmt.Println(pkgs)
	sts := parseNode(pkgs[*pkg])

	fmt.Printf("Structs: %#+v\n", sts)
}
