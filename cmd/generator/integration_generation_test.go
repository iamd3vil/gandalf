package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knadh/stuffbin"
)

func TestRunGeneratesCompilableComplexFieldTypes(t *testing.T) {
	dir := t.TempDir()
	source := strings.Join([]string{
		"package fixture",
		"",
		`import "time"`,
		"",
		"type User struct{}",
		"",
		"type Complex struct {",
		"\tMap map[string]int `validate:\"required min:1 max:5\"`",
		"\tArray [3]int `validate:\"min:1 max:4\"`",
		"\tPointer *User `validate:\"required\"`",
		"\tQualifiedPointer *time.Time `validate:\"required\"`",
		"\tPointerSlice []*User `validate:\"required min:1\"`",
		"\tQualifiedSlice []time.Time `validate:\"required min:1\"`",
		"\tPattern string `validate:\"regexp:'^\\\\d+:[\\\"x]+$'\"`",
		"}",
		"",
		"type Embedded struct { *User }",
	}, "\n")
	writeSourceFile(t, dir, "models.go", source)

	dest := filepath.Join(dir, "validations_gen.go")
	if err := run(newTemplatesFS(t), "fixture", dir, dest); err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}

	generated, err := ioutil.ReadFile(dest)
	if err != nil {
		t.Fatalf("read generated source: %v", err)
	}
	if !strings.Contains(string(generated), `regexp.MustCompile("^\\d+:[\"x]+$")`) {
		t.Fatalf("generated regexp was not safely quoted:\n%s", generated)
	}

	cmd := exec.Command("go", "test", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated package did not compile: %v\n%s\nGenerated source:\n%s", err, output, generated)
	}
}

func TestRunFormattingFailurePreservesDestination(t *testing.T) {
	sourceDir := t.TempDir()
	writeSourceFile(t, sourceDir, "models.go", "package fixture\n\ntype Record struct { Name string `validate:\"required\"` }\n")

	templateDir := t.TempDir()
	writeSourceFile(t, templateDir, "struct.tmpl", "this is not valid Go for package {{.Pkg}}")
	fs, err := stuffbin.NewLocalFS("/", templateDir+":/templates")
	if err != nil {
		t.Fatalf("create template filesystem: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "validations_gen.go")
	const sentinel = "existing output\n"
	if err := ioutil.WriteFile(dest, []byte(sentinel), 0600); err != nil {
		t.Fatalf("seed destination: %v", err)
	}

	if err := run(fs, "fixture", sourceDir, dest); err == nil || !strings.Contains(err.Error(), "formatting") {
		t.Fatalf("expected formatting error, got %v", err)
	}
	contents, err := ioutil.ReadFile(dest)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(contents) != sentinel {
		t.Fatalf("formatting failure changed destination to %q", contents)
	}
}

func TestRunUnknownConstraintPreservesDestination(t *testing.T) {
	dir := t.TempDir()
	writeSourceFile(t, dir, "models.go", "package fixture\n\ntype Record struct { Name string `validate:\"requried\"` }\n")
	dest := filepath.Join(t.TempDir(), "validations_gen.go")
	const sentinel = "existing output\n"
	if err := ioutil.WriteFile(dest, []byte(sentinel), 0644); err != nil {
		t.Fatalf("seed destination: %v", err)
	}

	err := run(newTemplatesFS(t), "fixture", dir, dest)
	if err == nil || !strings.Contains(err.Error(), "unknown constraint") || !strings.Contains(err.Error(), "Record") || !strings.Contains(err.Error(), "Name") {
		t.Fatalf("expected contextual unknown-constraint error, got %v", err)
	}
	contents, readErr := ioutil.ReadFile(dest)
	if readErr != nil {
		t.Fatalf("read destination: %v", readErr)
	}
	if string(contents) != sentinel {
		t.Fatalf("unknown constraint changed destination to %q", contents)
	}
}

func TestRegexpRejectsNonStringField(t *testing.T) {
	fields := []structField{{Name: "Count", Type: "int", Tag: "regexp:'^[0-9]+$'"}}
	_, err := getConstraints("Record", fields)
	if err == nil {
		t.Fatal("expected regexp on an int field to be rejected")
	}
	for _, context := range []string{"Record", "Count", "int"} {
		if !strings.Contains(err.Error(), context) {
			t.Fatalf("error %q does not include %q", err, context)
		}
	}
}
