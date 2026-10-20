package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	var out strings.Builder
	for _, args := range [][]string{nil, {"help"}, {"version"}, {"features"}} {
		out.Reset()
		if err := run(args, &out, false); err != nil {
			t.Errorf("%v: %v", args, err)
		}
	}
	if err := run([]string{"version"}, &out, false); err != nil || !strings.Contains(out.String(), "golang-vibe-starter") {
		t.Errorf("version: %q, %v", out.String(), err)
	}
	if err := run([]string{"nope"}, &out, false); err == nil {
		t.Error("an unknown command: no error")
	}
}

// TestNewRefuses checks what govibe new refuses, before it writes anything.
func TestNewRefuses(t *testing.T) {
	full := t.TempDir()
	if err := os.WriteFile(filepath.Join(full, "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]string{
		"no name, and no one to ask": {"-module", "github.com/acme/orders"},
		"a bad module":               {"-name", "orders", "-module", "orders"},
		"an unknown feature":         {"-name", "orders", "-module", "github.com/acme/orders", "-without", "nope"},
		"a directory in use":         {"-name", "orders", "-module", "github.com/acme/orders", full},
		"two directories":            {"-name", "orders", "-module", "github.com/acme/orders", "a", "b"},
	} {
		if err := run(append([]string{"new"}, args...), &strings.Builder{}, false); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

// TestEndToEnd makes services with govibe, as a user would, and runs their
// tests: one with everything, its resource renamed, and one with nothing
// optional. It takes minutes, and needs the network for the services'
// modules: make e2e runs it (GOVIBE_E2E=1), as CI does.
func TestEndToEnd(t *testing.T) {
	if os.Getenv("GOVIBE_E2E") == "" {
		t.Skip("GOVIBE_E2E is not set; make e2e runs this")
	}
	bin, err := filepath.Abs("bin/govibe")
	if err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]string{
		"everything, as categories": {"-resource", "category", "-resource-plural", "categories"},
		"nothing optional":          {"-without", "all"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), "shop")
			cmd := exec.Command(bin, append([]string{"new", "-name", "shop", "-module", "github.com/acme/shop", "-yes"}, append(args, dir)...)...)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("govibe new: %v\n%s", err, out)
			}
			for _, step := range [][]string{{"go", "vet", "./..."}, {"go", "test", "-count=1", "./..."}} {
				cmd := exec.Command(step[0], step[1:]...)
				cmd.Dir = dir
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Errorf("%s: %v\n%s", strings.Join(step, " "), err, out)
				}
			}
		})
	}
}
