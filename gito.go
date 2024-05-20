package gito

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type G struct {
	config *Config
}

func New(config *Config) *G {
	return &G{config: config}
}

func (g *G) Get(repo string) error {
	parsed, err := url.Parse(repo)
	if err != nil {
		return fmt.Errorf("gito: error parsing repo URL: %v", err)
	}
	repo = path.Join(parsed.Host, parsed.Path)

	// where repo will live in the PATH
	fullPath := filepath.Join(g.config.active.path[0], repo)

	err = os.MkdirAll(filepath.Dir(fullPath), 0755)
	if err != nil {
		return err
	}

	if exists, err := gitCloneAt(repo, fullPath); exists {
		return fmt.Errorf("gito: something already exists at %q", fullPath)
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
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("gito: error cloning repo: %v", err)
	}

	cmd = exec.Command("git", "submodule", "update", "--init", "--recursive")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = fullPath
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("gito: error updating submodules: %v", err)
	}

	return false, nil
}

func (g *G) Where(repo string) ([]string, error) {
	repo, _ = g.config.active.Alias(repo)
	path, ok := g.config.active.CustomPath(repo)
	if ok {
		return []string{path}, nil
	}

	return g.where(repo, true)
}

func (g *G) where(maybePath string, checkIsRepo bool) ([]string, error) {
	matches := map[string]struct{}{}
	mtx := sync.Mutex{}
	for _, dir := range g.config.active.path {
		newMatches, ok := in(maybePath, "", filepath.Join(dir), map[string]struct{}{}, checkIsRepo, 0, &mtx)
		if ok {
			for match := range newMatches {
				matches[match] = struct{}{}
			}
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("%q not found", maybePath)
	}

	paths := []string{}
	for match := range matches {
		paths = append(paths, match)
	}

	return paths, nil
}

// in is a recursive function that checks for repo inside of dir.
func in(repo, dir, so_far string, matches map[string]struct{}, check_is_repo bool, depth int, mtx *sync.Mutex) (map[string]struct{}, bool) {
	// limit recursion depth
	if depth == 3 {
		return matches, len(matches) > 0
	}

	full_path := filepath.Join(so_far, dir, repo)

	// check if the directory is a repository
	dir_is_repo := true
	if check_is_repo {
		dir_is_repo = isRepo(full_path)
	}

	if repo == dir && dir_is_repo {
		mtx.Lock()
		matches[full_path] = struct{}{}
		mtx.Unlock()
		return matches, true
	}

	// handle partial name matches
	f, err := os.Stat(full_path)
	if err == nil && f.IsDir() && dir_is_repo {
		mtx.Lock()
		matches[full_path] = struct{}{}
		mtx.Unlock()
		return matches, len(matches) > 0
	}

	files, err := os.ReadDir(filepath.Join(so_far, dir))
	if err != nil {
		return matches, len(matches) > 0
	}

	// collect results in a thread-local manner
	local_matches := make(map[string]struct{})
	wg := sync.WaitGroup{}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		wg.Add(1)
		go func(file fs.DirEntry) {
			defer wg.Done()
			new_matches, ok := in(repo, file.Name(), filepath.Join(so_far, dir), matches, check_is_repo, depth+1, mtx)
			if ok {
				mtx.Lock()
				for match := range new_matches {
					local_matches[match] = struct{}{}
				}
				mtx.Unlock()
			}
		}(file)
	}
	wg.Wait()

	// merge local matches into global matches
	mtx.Lock()
	for match := range local_matches {
		matches[match] = struct{}{}
	}
	mtx.Unlock()

	return matches, len(matches) > 0
}

// isRepo tests for the existence of a .git directory at dir.
func isRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return !os.IsNotExist(err)
}

func (g *G) URL(repo string) ([]string, error) {
	var paths = []string{"."}
	if repo != "." {
		var err error
		paths, err = g.Where(repo)
		if err != nil {
			return nil, err
		}
	}

	urls := []string{}
	errs := []error{}
	for _, path := range paths {
		url, err := getURL(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		urls = append(urls, url)
	}

	if len(errs) == len(paths) {
		return nil, fmt.Errorf("gito: no URLs found")
	}

	return urls, nil
}

func getURL(repo string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repo

	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error getting git remote for %q: %v", repo, err)
	}

	return extractURL(buf.String()), nil
}

func extractURL(url string) string {
	url = strings.TrimSpace(url)

	// removes prefix of url if it starts with ssh:// or git@
	url = strings.Replace(url, "git@", "", 1)
	url = strings.Replace(url, "ssh://", "", 1)
	url = strings.Replace(url, "http://", "", 1)
	url = strings.Replace(url, "https://", "", 1)
	url = strings.Replace(url, ":", "/", 1)
	url = strings.Replace(url, ".git", "", 1)

	return "https://" + url
}

func (g *G) Alias(from, to string) error {
	_, err := g.Where(to)
	if err != nil {
		return err
	}

	aliases := g.config.active.Aliases
	aliases[from] = to

	return g.config.Sync()
}

func (g *G) Set(name, loc string) error {
	if !isRepo(loc) {
		return fmt.Errorf("no repo @ %q", loc)
	}

	custom := g.config.active.Custom
	custom[name] = loc

	return g.config.Sync()
}

func (g *G) SetSelf(self string) error {
	_, err := g.where(self, false)
	if err != nil {
		return err
	}

	g.config.active.Self = self

	return g.config.Sync()
}

func (g *G) Self() (string, error) {
	self := g.config.active.Self
	if self == "" {
		return "", nil
	}

	where, err := g.where(self, false)
	if err != nil {
		return "", err
	}

	return where[0], nil
}
