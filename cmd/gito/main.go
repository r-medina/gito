package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/r-medina/gito"
)

var workspace string

func init() {
	flag.StringVar(&workspace, "w", "", "which workspace to use - setup ~/.config/gito/gito.json")

	flag.Parse()
}

func usage() {
	fmt.Printf(`usage: gito [<flags>] <command> [<args> ...]

Manage code intelligently.

See http://github.com/r-medina/gito for documentation.

Flags:
  -w WORKSPACE which workspace to use (defaults to first in config)

Commands:
  help
    show this message

  get <repo>
    download a repo

  where <repo>
    find out where repo lives

  url [<repo>|.]
    get the url of the repo (for web browsing) - can also pass no argument or "." for current directory

  alias <alias> <to>
    alias a name to something - eg "k8s" -> "github.com/kubernetes/kubernetes"

  set <alias> <location>
    for code living outside your configured path, tell gito where to find it

  set-self <location>
    configuring gito to use a default folder for your code

  self
    get location of self in config (default location to put your code)
`)
}

var cmds = map[string]func(_ *gito.G, args ...string){
	"get": func(g *gito.G, args ...string) {
		if len(args) != 1 {
			usage()
			os.Exit(1)
		}

		repo := args[0]
		exitIfErr(g.Get(repo), "getting %q", repo)
	},

	"where": func(g *gito.G, args ...string) {
		if len(args) != 1 {
			usage()
			os.Exit(1)
		}

		repo := args[0]
		fullPath, err := g.Where(repo)
		exitIfErr(err, "finding %q", repo)

		fmt.Println(fullPath)
	},

	"url": func(g *gito.G, args ...string) {
		if len(args) > 1 {
			usage()
			os.Exit(1)
		}
		path := "."
		if len(args) == 1 {
			path = args[0]
		}

		url, err := g.URL(path)
		exitIfErr(err, "getting URL for %q", path)

		fmt.Println(url)
	},

	"alias": func(g *gito.G, args ...string) {
		if len(args) != 2 {
			usage()
			os.Exit(1)
		}

		from, to := args[0], args[1]
		exitIfErr(g.Alias(from, to), "setting Alias %q -> %q", from, to)
	},

	"set": func(g *gito.G, args ...string) {
		if len(args) != 2 {
			usage()
			os.Exit(1)
		}

		name, loc := args[0], args[1]
		exitIfErr(g.Set(name, loc), "setting %q location to %q", name, loc)
	},

	"set-self": func(g *gito.G, args ...string) {
		if len(args) != 1 {
			usage()
			os.Exit(1)
		}

		self := args[0]
		exitIfErr(g.SetSelf(self), "setting self to %s", self)
	},

	"self": func(g *gito.G, args ...string) {
		if len(args) != 0 {
			usage()
			os.Exit(1)
		}

		self, err := g.Self()
		exitIfErr(err, "getting self")

		fmt.Println(self)
	},
}

func main() {
	if len(flag.Args()) < 1 {
		usage()
		os.Exit(1)
	}

	newConfig := false
	home, err := os.UserHomeDir()
	exitIfErr(err, "finding home directory")
	configName := filepath.Join(home, ".config/gito/gito.yaml")
	f, err := os.OpenFile(configName, os.O_SYNC|os.O_RDWR, 0622)
	if os.IsNotExist(err) {
		newConfig = true
		err = os.MkdirAll(filepath.Join(home, "/.config/gito"), 0755)
		exitIfErr(err, "making config dir")

		f, err = os.OpenFile(configName, os.O_SYNC|os.O_WRONLY|os.O_CREATE, 0622)
		exitIfErr(err, "making config file")
	} else {
		exitIfErr(err, "opening config file %q", configName)
	}

	config, err := gito.LoadConfig(f, newConfig, workspace)
	exitIfErr(err, "loading config @ %q", configName)

	g := gito.New(config)
	defer f.Close()

	cmd := cmds[flag.Args()[0]]
	if cmd == nil {
		usage()
		os.Exit(1)
	}
	cmd(g, flag.Args()[1:]...)
}

func exitIfErr(err error, format string, args ...interface{}) {
	if err == nil {
		return
	}

	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintf(os.Stderr, ": %v\n", err)
	os.Exit(1)

}
