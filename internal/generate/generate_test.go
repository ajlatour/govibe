package generate

import (
	"slices"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	ok := Options{Dir: "x", Name: "orders", Module: "github.com/acme/orders", Resource: "category", ResourcePlural: "categories", Without: []string{"perf"}}
	if err := ok.Check(); err != nil {
		t.Errorf("%+v: %v", ok, err)
	}
	for name, mod := range map[string]func(*Options){
		"a name with capitals":   func(o *Options) { o.Name = "Orders" },
		"no module":              func(o *Options) { o.Module = "" },
		"a resource with a dash": func(o *Options) { o.Resource = "line-item" },
		"a plural alone":         func(o *Options) { o.Resource = "" },
		"an unknown feature":     func(o *Options) { o.Without = []string{"perf", "nope"} },
		"no directory":           func(o *Options) { o.Dir = "" },
	} {
		o := ok
		mod(&o)
		if err := o.Check(); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestInitArgs(t *testing.T) {
	o := Options{Name: "orders", Module: "github.com/acme/orders", Resource: "category", ResourcePlural: "categories", Without: []string{"perf", "release"}}
	want := "-name orders -module github.com/acme/orders -without perf,release -resource category -resource-plural categories"
	if got := strings.Join(o.InitArgs(), " "); got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
	if got := (Options{Name: "x", Module: "m/x"}).InitArgs(); !slices.Equal(got, []string{"-name", "x", "-module", "m/x"}) {
		t.Errorf("got %v", got)
	}
}
