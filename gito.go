package gito

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type G struct {
	config *Config
}

func New(config *Config) *G {
	return &G{config: config}
}

func (g *G) Get(repo string) error {
	// where repo will live in the PATH
	fullPath := filepath.Join(g.config.active.path[0], repo)

	err := os.MkdirAll(filepath.Dir(fullPath), 0755)
	if err != nil {
		return err
	}

	if exists, err := gitCloneAt(repo, fullPath); exists {
		return fmt.Errorf("something already exists at %q", fullPath)
	} else if err != nil {
		return err
	}

	return nil
}

func gitCloneAt(repo, fullPath string) (bool, error) {
	_, err := os.Stat(fullPath)
	if !os.IsNotExist(err) {
		return true, nil
	}

	gitRepo := fmt.Sprintf("https://%s.git", repo) // simpler than ssh
	cmd := exec.Command("git", "clone", "--", gitRepo, fullPath)
	buf := &bytes.Buffer{}
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("error cloning repo: %v, stderr: %q", err, buf.String())
	}

	cmd = exec.Command("git", "submodule", "update", "--init", "--recursive")
	buf.Reset()
	cmd.Stderr = buf
	cmd.Dir = fullPath
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("error updating submodules: %v, stderr: %s", err, buf.String())
	}

	return false, nil
}

func (g *G) Where(repo string) (string, error) {
	repo, _ = g.config.active.Alias(repo)
	path, ok := g.config.active.CustomPath(repo)
	if ok {
		return path, nil
	}

	for _, dir := range g.config.active.path {
		fullPath, ok := in(repo, filepath.Join(dir), "", 0)
		if ok {
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("%q not found", repo)
}

// in is a recursive function that checks for repo inside of dir.
func in(repo, dir, soFar string, depth int) (string, bool) {
	// don't check git directories
	if dir == ".git" {
		return "", false
	}

	fullPath := filepath.Join(soFar, dir, repo)

	// found it

	dirIsRepo := isRepo(fullPath)

	if repo == dir && dirIsRepo {
		return fullPath, true
	}

	// in case repo is a partial name (ie r-medina/gito)
	f, err := os.Stat(fullPath)
	if err == nil && dirIsRepo {
		return fullPath, f.IsDir() // make sure we're not getting a file
	}

	// don't want to go past repositories
	if depth == 3 {
		return "", false
	}

	files, err := ioutil.ReadDir(filepath.Join(soFar, dir))
	if err != nil {
		return "", false
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		fullPath, ok := in(repo, file.Name(), filepath.Join(soFar, dir), depth+1)
		if ok {
			return fullPath, true
		}
	}

	return "", false
}

// isRepo tests for the existence of a .git directory at dir.
func isRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return !os.IsNotExist(err)
}

func (g *G) URL(repo string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")

	if repo != "." {
		fullPath, err := g.Where(repo)
		if err != nil {
			return "", err
		}
		cmd.Dir = fullPath
	}

	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error getting git remote for %q", repo)
	}

	url := buf.String()
	url = strings.TrimSpace(url)

	if strings.HasPrefix(url, "git@") {
		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
		buf.Reset()
		buf.WriteString("https://")
		buf.WriteString(url)
		url = buf.String()
	}

	// when origin is specified with http, all we need to do is trim suffix

	url = strings.TrimSuffix(url, ".git")

	return url, nil
}

func (g *G) Alias(from, to string) error {
	aliases := g.config.active.Aliases
	// no nil check because config load does that
	g.config.active.Aliases = aliases

	aliases[from] = to

	return g.config.Sync()
}

func (g *G) Set(name, loc string) error {
	custom := g.config.active.Custom
	custom[name] = loc

	return g.config.Sync()
}
