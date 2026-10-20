// Package generate makes a service from the template: unpacks it, runs its
// own make init with the choices made, and commits the result.
package generate

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ajlatour/govibe/internal/template"
)

// Options are the choices a service is made with.
type Options struct {
	// Dir is where it goes: a directory that's empty, or doesn't exist yet.
	Dir string
	// Name names the service: its images, compose project and docs.
	Name string
	// Module is its Go module path.
	Module string
	// Resource and ResourcePlural name its first resource; empty keeps the
	// template's example, item (items).
	Resource, ResourcePlural string
	// Without are the template's optional features to leave out.
	Without []string
	// NoCommit leaves the result uncommitted.
	NoCommit bool
}

// What the template's make init accepts, checked here too so a mistake is
// told before anything is written.
var (
	nameRE     = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	moduleRE   = regexp.MustCompile(`^[a-z0-9.-]+(/[A-Za-z0-9._~-]+)+$`)
	resourceRE = regexp.MustCompile(`^[a-z][a-z0-9]{0,30}$`)
)

// ValidName checks a service's name.
func ValidName(s string) error {
	if !nameRE.MatchString(s) {
		return errors.New("lower-case letters, digits and dashes, starting with a letter, e.g. orders")
	}
	return nil
}

// ValidModule checks a Go module path.
func ValidModule(s string) error {
	if !moduleRE.MatchString(s) {
		return errors.New("a module path, e.g. github.com/acme/orders")
	}
	return nil
}

// ValidResource checks a resource's name, or its plural; empty is the
// template's.
func ValidResource(s string) error {
	if s != "" && !resourceRE.MatchString(s) {
		return errors.New("lower-case letters and digits, starting with a letter, e.g. order")
	}
	return nil
}

// Check reports what's wrong with o, if anything.
func (o Options) Check() error {
	var errs []error
	if err := ValidName(o.Name); err != nil {
		errs = append(errs, fmt.Errorf("name %q: %w", o.Name, err))
	}
	if err := ValidModule(o.Module); err != nil {
		errs = append(errs, fmt.Errorf("module %q: %w", o.Module, err))
	}
	for _, r := range []string{o.Resource, o.ResourcePlural} {
		if err := ValidResource(r); err != nil {
			errs = append(errs, fmt.Errorf("resource %q: %w", r, err))
		}
	}
	if o.ResourcePlural != "" && o.Resource == "" {
		errs = append(errs, errors.New("a resource plural without a resource name"))
	}
	features, err := template.Features()
	if err != nil {
		return err
	}
	for _, w := range o.Without {
		known := false
		for _, f := range features {
			known = known || f.Name == w
		}
		if !known {
			errs = append(errs, fmt.Errorf("feature %q: the template has none by that name (govibe features lists them)", w))
		}
	}
	if o.Dir == "" {
		errs = append(errs, errors.New("no directory to make it in"))
	}
	return errors.Join(errs...)
}

// InitArgs are the arguments to the template's make init (initservice).
func (o Options) InitArgs() []string {
	args := []string{"-name", o.Name, "-module", o.Module}
	if len(o.Without) > 0 {
		args = append(args, "-without", strings.Join(o.Without, ","))
	}
	if o.Resource != "" {
		args = append(args, "-resource", o.Resource)
	}
	if o.ResourcePlural != "" {
		args = append(args, "-resource-plural", o.ResourcePlural)
	}
	return args
}

// Run makes the service: the template unpacked into o.Dir, a git repository
// of it, its make init run, and, unless o.NoCommit, its first commit. What
// the tools print goes to out.
func Run(o Options, out io.Writer) error {
	if err := o.Check(); err != nil {
		return err
	}
	for _, tool := range []string{"git", "go"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s isn't installed, and making a service needs it", tool)
		}
	}
	if err := template.Unpack(o.Dir); err != nil {
		return err
	}
	steps := [][]string{
		{"git", "init", "-q"},
		// make init rewrites the files under version control.
		{"git", "add", "-A"},
		append([]string{"go", "run", "./cmd/initservice"}, o.InitArgs()...),
	}
	commit := !o.NoCommit
	if commit && !gitIdentity(o.Dir) {
		commit = false
		fmt.Fprintln(out, "git has no user.name and user.email here, so the result isn't committed: set them, then git add -A && git commit.")
	}
	if commit {
		steps = append(steps, []string{"git", "add", "-A"},
			[]string{"git", "commit", "-q", "-m", fmt.Sprintf("Start %s from golang-vibe-starter %s", o.Name, template.Ref())})
	}
	for _, s := range steps {
		cmd := exec.Command(s[0], s[1:]...)
		cmd.Dir = o.Dir
		cmd.Stdout, cmd.Stderr = out, out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", strings.Join(s, " "), err)
		}
	}
	return nil
}

// gitIdentity reports whether git, in dir, knows who commits.
func gitIdentity(dir string) bool {
	for _, key := range []string{"user.name", "user.email"} {
		cmd := exec.Command("git", "config", key)
		cmd.Dir = dir
		if out, err := cmd.Output(); err != nil || strings.TrimSpace(string(out)) == "" {
			return false
		}
	}
	return true
}

// DirFree reports whether dir is empty, or doesn't exist.
func DirFree(dir string) bool {
	entries, err := os.ReadDir(dir)
	return errors.Is(err, os.ErrNotExist) || (err == nil && len(entries) == 0)
}
