package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knadh/stuffbin"
)

// newTemplatesFS builds a real stuffbin.FileSystem backed by the templates
// directory checked into the repository, mirroring what initFileSystem
// would provide from the packed binary at runtime. This lets tests exercise
// generateCodeForStructs (and therefore run) end-to-end without needing a
// stuffbin-packed test binary.
func newTemplatesFS(t *testing.T) stuffbin.FileSystem {
	t.Helper()

	fs, err := stuffbin.NewLocalFS(".", "../../templates:/templates")
	if err != nil {
		t.Fatalf("couldn't build local templates fs: %v", err)
	}
	return fs
}

// writeSourceFile writes a Go source file into dir so that parser.ParseDir
// picks it up.
func writeSourceFile(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := ioutil.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("couldn't write source fixture: %v", err)
	}
}

// dirEntryNames returns the sorted base names of every entry in dir.
func dirEntryNames(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("couldn't read dir %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// assertNoTempFiles fails the test if any stray atomic-write temp file is
// left behind in dir.
func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()

	for _, name := range dirEntryNames(t, dir) {
		if strings.HasPrefix(name, ".tmp-") {
			t.Errorf("leftover temp file not cleaned up: %s", name)
		}
	}
}

const validSourceFixture = `package foo

type Sample struct {
	Name string ` + "`validate:\"required\"`" + `
}
`

// invalidRegexSourceFixture declares a struct with an unterminated regex
// group so that regexp.Compile fails during code generation.
const invalidRegexSourceFixture = `package foo

type Sample struct {
	Name string ` + "`validate:\"regexp:'('\"`" + `
}
`

func TestRun_UnknownPackageLeavesDestinationUntouched(t *testing.T) {
	srcDir := t.TempDir()
	writeSourceFile(t, srcDir, "sample.go", validSourceFixture)

	destDir := t.TempDir()
	dest := filepath.Join(destDir, "out.go")
	const sentinel = "existing generated content\n"
	if err := ioutil.WriteFile(dest, []byte(sentinel), 0640); err != nil {
		t.Fatalf("couldn't seed destination file: %v", err)
	}

	// fs is never touched because the unknown package error must surface
	// before AST traversal or any template/codegen work happens.
	err := run(nil, "bar", srcDir, dest)
	if err == nil {
		t.Fatal("expected an error for an unknown package, got nil")
	}
	if !strings.Contains(err.Error(), "bar") {
		t.Errorf("error %q should mention the requested package %q", err, "bar")
	}
	if !strings.Contains(err.Error(), "foo") {
		t.Errorf("error %q should mention the found package %q", err, "foo")
	}

	got, rerr := ioutil.ReadFile(dest)
	if rerr != nil {
		t.Fatalf("destination file should still exist: %v", rerr)
	}
	if string(got) != sentinel {
		t.Errorf("destination content changed: got %q, want %q", got, sentinel)
	}

	info, serr := os.Stat(dest)
	if serr != nil {
		t.Fatalf("couldn't stat destination: %v", serr)
	}
	if info.Mode().Perm() != 0640 {
		t.Errorf("destination mode changed: got %v, want %v", info.Mode().Perm(), os.FileMode(0640))
	}

	assertNoTempFiles(t, destDir)

	names := dirEntryNames(t, destDir)
	if len(names) != 1 {
		t.Errorf("expected only the destination file in %s, found %v", destDir, names)
	}
}

func TestRun_GenerationFailureLeavesDestinationUntouched(t *testing.T) {
	srcDir := t.TempDir()
	writeSourceFile(t, srcDir, "sample.go", invalidRegexSourceFixture)

	destDir := t.TempDir()
	dest := filepath.Join(destDir, "out.go")
	const sentinel = "existing generated content\n"
	if err := ioutil.WriteFile(dest, []byte(sentinel), 0640); err != nil {
		t.Fatalf("couldn't seed destination file: %v", err)
	}

	fs := newTemplatesFS(t)

	err := run(fs, "foo", srcDir, dest)
	if err == nil {
		t.Fatal("expected an error for an invalid regexp constraint, got nil")
	}

	got, rerr := ioutil.ReadFile(dest)
	if rerr != nil {
		t.Fatalf("destination file should still exist: %v", rerr)
	}
	if string(got) != sentinel {
		t.Errorf("destination content changed: got %q, want %q", got, sentinel)
	}

	info, serr := os.Stat(dest)
	if serr != nil {
		t.Fatalf("couldn't stat destination: %v", serr)
	}
	if info.Mode().Perm() != 0640 {
		t.Errorf("destination mode changed: got %v, want %v", info.Mode().Perm(), os.FileMode(0640))
	}

	assertNoTempFiles(t, destDir)
}

func TestRun_SuccessPreservesExistingDestinationMode(t *testing.T) {
	srcDir := t.TempDir()
	writeSourceFile(t, srcDir, "sample.go", validSourceFixture)

	destDir := t.TempDir()
	dest := filepath.Join(destDir, "out.go")
	if err := ioutil.WriteFile(dest, []byte("stale content\n"), 0640); err != nil {
		t.Fatalf("couldn't seed destination file: %v", err)
	}

	fs := newTemplatesFS(t)

	if err := run(fs, "foo", srcDir, dest); err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}

	got, rerr := ioutil.ReadFile(dest)
	if rerr != nil {
		t.Fatalf("couldn't read generated destination: %v", rerr)
	}
	if !strings.Contains(string(got), "package foo") {
		t.Errorf("generated output missing expected package clause: %q", got)
	}
	if !strings.Contains(string(got), "func (s *Sample) Validate() error") {
		t.Errorf("generated output missing expected Validate method: %q", got)
	}

	info, serr := os.Stat(dest)
	if serr != nil {
		t.Fatalf("couldn't stat destination: %v", serr)
	}
	if info.Mode().Perm() != 0640 {
		t.Errorf("existing destination mode not preserved: got %v, want %v", info.Mode().Perm(), os.FileMode(0640))
	}

	assertNoTempFiles(t, destDir)
}

func TestRun_SuccessNewFileGetsDefaultMode(t *testing.T) {
	dir := t.TempDir()
	writeSourceFile(t, dir, "sample.go", validSourceFixture)

	dest := filepath.Join(dir, "out.go")
	fs := newTemplatesFS(t)

	if err := run(fs, "foo", dir, dest); err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}

	info, serr := os.Stat(dest)
	if serr != nil {
		t.Fatalf("expected destination file to be created: %v", serr)
	}
	if info.Mode().Perm() != defaultFileMode {
		t.Errorf("new file mode = %v, want %v", info.Mode().Perm(), os.FileMode(defaultFileMode))
	}

	assertNoTempFiles(t, dir)
}

func TestAtomicWriteFile_CleansUpTempOnRenameFailure(t *testing.T) {
	dir := t.TempDir()

	// A destination that is itself a directory makes the final os.Rename
	// fail, letting us confirm the temp file created along the way is
	// removed rather than left behind.
	dest := filepath.Join(dir, "out.go")
	if err := os.Mkdir(dest, 0755); err != nil {
		t.Fatalf("couldn't create directory destination: %v", err)
	}

	err := atomicWriteFile(dest, []byte("package foo\n"))
	if err == nil {
		t.Fatal("expected an error when destination is a directory, got nil")
	}

	names := dirEntryNames(t, dir)
	if len(names) != 1 || names[0] != "out.go" {
		t.Errorf("expected only the destination directory to remain in %s, found %v", dir, names)
	}
}
