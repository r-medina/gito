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
	flag.StringVar(&workspace, "workspace", "", "which workspace to use - setup ~/.config/gito/gito.json")
	if workspace == "" {
		flag.StringVar(&workspace, "w", "", "which workspace to use - setup ~/.config/gito/gito.json")
	}
}

func usage() {
	fmt.Printf(`usage: gito [<flags>] <command> [<args> ...]

Manage code intelligently.

See http://github.com/r-medina/gito for documentation.

Flags:
  --workspace=WORKSPACE which workspace to use (defaults to first in config)
    -w

Commands:
  help
    show this message

  get <repo>
    download a repo

  where <repo>
    find out where repo lives

  url <repo>
    get the url of the repo (for web browsing)
`)
}

var cmds = map[string]func(_ *gito.G, args ...string){
	"get": func(g *gito.G, args ...string) {
		if len(args) != 1 {
			usage()
			os.Exit(1)
		}

		if err := g.Get(args[0]); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},

	"where": func(g *gito.G, args ...string) {
		if len(args) != 1 {
			usage()
			os.Exit(1)
		}

		fullPath, err := g.Where(args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(fullPath)
	},

	"url": func(g *gito.G, args ...string) {
		if len(args) != 1 {
			usage()
			os.Exit(1)
		}

		url, err := g.URL(args[0])
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		fmt.Println(url)
	},
}

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}

	newConfig := false
	home, err := os.UserHomeDir()
	fname := filepath.Join(home, ".config/gito/gito.yaml")
	f, err := os.OpenFile(fname, os.O_SYNC|os.O_RDONLY, 0622)
	if os.IsNotExist(err) {
		newConfig = true
		if err = os.MkdirAll(filepath.Join(home, "/.config/gito"), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "error making config dir: %v", err)
			os.Exit(1)
		}

		f, err = os.OpenFile(fname, os.O_SYNC|os.O_WRONLY|os.O_CREATE, 0622)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error making config file: %v", err)
			os.Exit(1)
		}
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "error opening config: %v", err)
		os.Exit(1)
	}

	config, err := gito.LoadConfig(f, newConfig, workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	g := gito.New(config)
	defer f.Close()
	cmds[os.Args[1]](g, os.Args[2:]...)
}
