package gito

import (
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

func doGitClone(repo, fullPath string) error {
	gitRepo := fmt.Sprintf("https://%s.git", repo)
	cmd := exec.Command("git", "clone", "--", gitRepo, fullPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// TODO
		return err
	}

	// cmd = exec.Command("git", "submodule", "update", "--init", "--recursive")
	// cmd.Path = fullPath
	// if err := cmd.Run(); err != nil {
	// 	// TODO
	// 	return err
	// }

	return nil
}
