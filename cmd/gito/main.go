package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
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
		paths, err := g.Where(repo)
		exitIfErr(err, "finding %q", repo)

		prompt := promptui.Select{
			Label: "select a repo",
			Items: func() []string {
				// trims directories ending at src/
				// ie /home/rmedina/go/src/github.com/r-medina/gito
				// becomes github.com/r-medina/gito

				for i, path := range paths {
					if !strings.Contains(path, "src") {
						continue
					}

					paths[i] = path[strings.Index(path, "src")+4:]
				}

				return paths
			}(),
			Stdout: os.Stderr, // so it doesn't get lost in redirects
		}
		_, path, err := prompt.Run()
		exitIfErr(err, "prompting for repo")

		fmt.Println(path)
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

		urls, err := g.URL(path)
		exitIfErr(err, "getting URL for %q", path)

		if len(urls) == 1 {
			fmt.Println(urls[0])
			return
		}

		prompt := promptui.Select{
			Label: "select a repo",
			Items: func() []string {
				for i, url := range urls {
					// parses url and removes the protocol

					urls[i] = url[strings.Index(url, "://")+3:]
				}

				return urls
			}(),
			Stdout: os.Stderr, // so it doesn't get lost in redirects
		}
		_, url, err := prompt.Run()
		exitIfErr(err, "prompting for repo")

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

	config, err := gito.LoadConfig(workspace)
	exitIfErr(err, "loading config")
	defer config.Close()

	g := gito.New(config)

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
