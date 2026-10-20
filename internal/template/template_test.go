package template

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type entry struct {
	name string
	mode int64
	typ  byte
	body string
}

func tarball(t *testing.T, entries ...entry) *bytes.Reader {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Mode: e.mode, Typeflag: e.typ, Size: int64(len(e.body))}
		if e.typ == tar.TypeSymlink {
			h.Linkname, h.Size = "/etc/passwd", 0
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if e.typ == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(b.Bytes())
}

func TestUnpack(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new")
	err := unpack(tarball(t,
		entry{name: "pax_global_header", typ: tar.TypeXGlobalHeader},
		entry{name: "scripts/", typ: tar.TypeDir, mode: 0o755},
		entry{name: "scripts/api", typ: tar.TypeReg, mode: 0o755, body: "#!/bin/sh\n"},
		entry{name: "README.md", typ: tar.TypeReg, mode: 0o644, body: "# x\n"},
	), dir)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "README.md")); err != nil || string(b) != "# x\n" {
		t.Errorf("README.md: %q, %v", b, err)
	}
	if info, err := os.Stat(filepath.Join(dir, "scripts/api")); err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Errorf("scripts/api: %v, %v; want it executable", info, err)
	}
}

func TestUnpackRefuses(t *testing.T) {
	for name, e := range map[string]entry{
		"a path out of the directory": {name: "../escape", typ: tar.TypeReg, body: "x"},
		"an absolute path":            {name: "/tmp/escape", typ: tar.TypeReg, body: "x"},
		"a link":                      {name: "link", typ: tar.TypeSymlink},
	} {
		dir := filepath.Join(t.TempDir(), "new")
		if err := unpack(tarball(t, e), dir); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	full := t.TempDir()
	if err := os.WriteFile(filepath.Join(full, "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := unpack(tarball(t), full); err == nil || !strings.Contains(err.Error(), "isn't empty") {
		t.Errorf("a directory with something in it: %v", err)
	}
}

// TestSnapshot checks the embedded template is there, unpacks, and lists
// its features.
func TestSnapshot(t *testing.T) {
	if Ref() == "" {
		t.Error("no REF")
	}
	features, err := Features()
	if err != nil || len(features) == 0 {
		t.Fatalf("features: %v, %v", features, err)
	}
	dir := filepath.Join(t.TempDir(), "svc")
	if err := Unpack(dir); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"go.mod", "cmd/initservice/main.go", "Makefile"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("the snapshot has no %s: %v", f, err)
		}
	}
}
