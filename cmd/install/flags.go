package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Options struct {
	CoreOnly       bool
	WithBusiness   bool
	NoTools        bool
	NoRoles        bool
	Platforms      string
	Location       string
	Link           bool
	DryRun         bool
	NonInteractive bool
	SourceDir      string
}

func ParseFlags() Options {
	var opts Options
	flag.BoolVar(&opts.CoreOnly, "core-only", false, "install only core/")
	flag.BoolVar(&opts.WithBusiness, "with-business", false, "include business/ module")
	flag.BoolVar(&opts.NoTools, "no-tools", false, "skip tools/ module")
	flag.BoolVar(&opts.NoRoles, "no-roles", false, "skip roles/ module")
	flag.StringVar(&opts.Platforms, "platforms", "", "comma-separated platform list (default: auto-detect)")
	flag.StringVar(&opts.Location, "location", expandPath("~/.gpowers"), "install location")
	flag.BoolVar(&opts.Link, "link", false, "symlink source repo (dev mode)")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "print plan, do not execute")
	flag.BoolVar(&opts.NonInteractive, "non-interactive", false, "skip prompts (CI mode)")
	flag.StringVar(&opts.SourceDir, "source-dir", "", "override source directory")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: install [options]\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	return opts
}

func (o Options) Modules() []string {
	if o.CoreOnly {
		return []string{"core"}
	}
	mods := []string{"core"}
	if !o.NoRoles {
		mods = append(mods, "roles")
	}
	if !o.NoTools {
		mods = append(mods, "tools")
	}
	if o.WithBusiness {
		mods = append(mods, "business")
	}
	return mods
}

func (o Options) PlatformList() []string {
	if o.Platforms == "" {
		return nil
	}
	parts := strings.Split(o.Platforms, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
