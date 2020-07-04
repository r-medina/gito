package gito

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

type G struct {
	path []string
}

func New(path string) *G {
	paths := []string{}
	for _, dir := range filepath.SplitList(path) {
		paths = append(paths, filepath.Join(dir, "src"))

	}

	return &G{path: paths}
}

func (g *G) Get(repo string) error {
	// where repo will live in the PATH
	fullPath := filepath.Join(g.path[0], repo)

	err := os.MkdirAll(fullPath, 0755)
	if err != nil {
		// TODO
		return err
	}

	if err := doGitClone(repo, fullPath); err != nil {
		// TODO
		return err
	}

	return nil
}

func (g *G) Where(repo string) (string, error) {
	for _, dir := range g.path {
		fullPath := filepath.Join(dir, repo)
		_, err := os.Stat(fullPath)
		if err == nil {
			return fullPath, nil
		}
	}

	// if user passed name of repo without owner

	for _, dir := range g.path {
		fullPath, ok := in(repo, dir, "", 0)
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

	// found it
	if repo == dir {
		return filepath.Join(soFar, repo), true
	}

	// in case repo is a partial name (ie r-medina/gito)
	fullPath := filepath.Join(soFar, dir, repo)
	_, err := os.Stat(fullPath)
	if err == nil {
		return fullPath, true
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
			return fullPath, ok
		}
	}

	return "", false
}

func doGitClone(repo, fullPath string) error {
	gitRepo := fmt.Sprintf("https://%s.git", repo) // simpler than ssh
	cmd := exec.Command("git", "clone", "--", gitRepo, fullPath)
	buf := &bytes.Buffer{}
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		// TODO
		return fmt.Errorf("error cloning repo: %v, stderr: %q", err, buf.String())
	}

	cmd = exec.Command("git", "submodule", "update", "--init", "--recursive")
	buf.Reset()
	cmd.Stderr = buf
	cmd.Dir = fullPath
	if err := cmd.Run(); err != nil {
		// TODO
		return fmt.Errorf("error updating submodules: %v, stdout: %s", err, buf.String())
	}

	return nil
}
