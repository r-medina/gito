package main

import (
	"fmt"
	"go/build"
	"os"

	"github.com/r-medina/gito"
)

func usage() {
	fmt.Printf(`usage: gito <command> [<args> ...]

Manage code intelligently.

Commands:
  help
    show this message

  get <repo>
    download a repo into your gopath (eg github.com/r-medina/gito)

  where <repo>
    find out where repo lives
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

	g := gito.New(build.Default.GOPATH)
	cmds[os.Args[1]](g, os.Args[2:]...)
}
