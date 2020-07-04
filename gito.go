package gito

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type G struct {
	path []string
}

func New(path string) *G {
	return &G{path: filepath.SplitList(path)}
}

func (g *G) Get(repo string) error {
	// where repo will live in the PATH
	fullPath := filepath.Join(g.path[0], "src", repo)

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
		fullPath := filepath.Join(dir, "src", repo)
		f, err := os.Open(fullPath)
		f.Close()
		if err == nil {
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("%q not found", repo)
}

func doGitClone(repo, fullPath string) error {
	gitRepo := fmt.Sprintf("https://%s.git", repo)
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
