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
			fmt.Println(usage)
			os.Exit(1)
		}

		if err := g.Get(args[0]); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},

	"where": func(g *gito.G, args ...string) {
		if len(args) != 1 {
			fmt.Println(usage)
			os.Exit(1)
		}

	},
}

func main() {
	g := gito.New(build.Default.GOPATH)

	fmt.Println(os.Args)
	cmds[os.Args[1]](g, os.Args[2:]...)
}
