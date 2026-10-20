// Command govibe makes a new Go service from the golang-vibe-starter
// template: it asks what the service is called, its module, its first
// resource, and which of the template's optional parts it should have, then
// unpacks the template, runs the template's own make init with those
// choices, and commits the result.
//
//	govibe new                  # asks
//	govibe new -name orders -module github.com/acme/orders -resource order \
//	    -without perf,release -yes
//	govibe features             # what can be left out
//	govibe version
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/huh/v2"

	"github.com/ajlatour/govibe/internal/generate"
	"github.com/ajlatour/govibe/internal/template"
)

// version is govibe's own, set at build time (-ldflags "-X main.version=...").
var version = "devel"

func main() {
	if err := run(os.Args[1:], os.Stdout, isTerminal(os.Stdin)); err != nil {
		fmt.Fprintln(os.Stderr, "govibe:", err)
		os.Exit(1)
	}
}

const usage = `govibe makes a new Go service from the golang-vibe-starter template.

  govibe new [flags] [dir]   make one (it asks what's not given)
  govibe features            the template's optional parts
  govibe version             govibe's version, and the template's

govibe new -h lists the flags.`

func run(args []string, out io.Writer, interactive bool) error {
	if len(args) == 0 {
		fmt.Fprintln(out, usage)
		return nil
	}
	switch args[0] {
	case "new":
		return runNew(args[1:], out, interactive)
	case "features":
		features, err := template.Features()
		if err != nil {
			return err
		}
		for _, f := range features {
			fmt.Fprintf(out, "%-12s %s\n", f.Name, f.Summary)
		}
		return nil
	case "version":
		fmt.Fprintf(out, "govibe %s, golang-vibe-starter %s\n", version, template.Ref())
		return nil
	case "-h", "-help", "--help", "help":
		fmt.Fprintln(out, usage)
		return nil
	}
	return fmt.Errorf("no command %q\n\n%s", args[0], usage)
}

func runNew(args []string, out io.Writer, interactive bool) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(out)
	var o generate.Options
	fs.StringVar(&o.Name, "name", "", "the service's name: lower-case letters, digits and dashes, e.g. orders")
	fs.StringVar(&o.Module, "module", "", "its Go module path, e.g. github.com/acme/orders")
	fs.StringVar(&o.Resource, "resource", "", "its first resource, singular, e.g. order (default: the template's example, item)")
	fs.StringVar(&o.ResourcePlural, "resource-plural", "", "its plural, if it isn't the name with an s")
	without := fs.String("without", "", "the template's optional parts to leave out, comma-separated (govibe features lists them), or all")
	yes := fs.Bool("yes", false, "ask nothing: take the flags as they are")
	fs.BoolVar(&o.NoCommit, "no-commit", false, "leave the result uncommitted")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("one directory, not %d", fs.NArg())
	}
	features, err := template.Features()
	if err != nil {
		return err
	}
	if *without == "all" {
		for _, f := range features {
			o.Without = append(o.Without, f.Name)
		}
	} else if *without != "" {
		o.Without = strings.Split(*without, ",")
	}

	if interactive && !*yes {
		if err := ask(&o, features, *without != ""); err != nil {
			return err
		}
	}
	o.Dir = fs.Arg(0)
	if o.Dir == "" {
		o.Dir = o.Name
	}
	if err := o.Check(); err != nil {
		return err
	}
	if !generate.DirFree(o.Dir) {
		return fmt.Errorf("%s already exists, and isn't empty: give a new directory", o.Dir)
	}
	fmt.Fprintf(out, "Making %s (%s) in %s, from golang-vibe-starter %s...\n", o.Name, o.Module, o.Dir, template.Ref())
	if err := generate.Run(o, out); err != nil {
		return err
	}
	abs, _ := filepath.Abs(o.Dir)
	fmt.Fprintf(out, "\nDone: %s. Next:\n\n  cd %s\n  make dev        # the API on :8080, nothing else to start\n  make check      # the tests\n", abs, o.Dir)
	return nil
}

// ask fills in what o lacks, and lets the features be picked, unless
// -without already said.
func ask(o *generate.Options, features []template.Feature, featuresGiven bool) error {
	var fields []huh.Field
	if o.Name == "" {
		fields = append(fields, huh.NewInput().Title("The service's name").
			Description("Lower-case letters, digits and dashes: it names the images, the compose project and the docs.").
			Placeholder("orders").Value(&o.Name).Validate(generate.ValidName))
	}
	if o.Module == "" {
		fields = append(fields, huh.NewInput().Title("Its Go module path").
			Placeholder("github.com/acme/orders").Value(&o.Module).Validate(generate.ValidModule))
	}
	if o.Resource == "" {
		fields = append(fields, huh.NewInput().Title("Its first resource, singular").
			Description("The template's example, /items, renamed: order makes /orders. Empty keeps /items.").
			Placeholder("order").Value(&o.Resource).Validate(generate.ValidResource),
			huh.NewInput().Title("Its plural, if it isn't the name with an s").
				Description("categories, say, for category. Empty adds an s.").
				Value(&o.ResourcePlural).Validate(generate.ValidResource))
	}
	var keep []string
	if !featuresGiven {
		var opts []huh.Option[string]
		for _, f := range features {
			opts = append(opts, huh.NewOption(f.Name+": "+f.Summary, f.Name).Selected(true))
		}
		fields = append(fields, huh.NewMultiSelect[string]().Title("What should it have?").
			Description("Each is optional: what's left unticked is left out, code, docs and tests alike.").
			Options(opts...).Value(&keep))
	}
	if len(fields) == 0 {
		return nil
	}
	// ACCESSIBLE=1 asks line by line, for a screen reader (and a script),
	// rather than drawing a form.
	form := huh.NewForm(huh.NewGroup(fields...)).WithAccessible(os.Getenv("ACCESSIBLE") != "")
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return errors.New("stopped: nothing was made")
		}
		return err
	}
	if !featuresGiven {
		o.Without = nil
		for _, f := range features {
			if !slices.Contains(keep, f.Name) {
				o.Without = append(o.Without, f.Name)
			}
		}
	}
	return nil
}

// isTerminal reports whether f is a terminal a person can answer at.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
