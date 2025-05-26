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
	"regexp"
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

func in(repo, dir, soFar string, matches map[string]struct{}, checkIsRepo bool, depth int, mtx *sync.Mutex) (map[string]struct{}, bool) {
	// limit recursion depth - gitlab does let you do sub directories, so
	// increased from 3 to 4 to support them
	if depth == 4 {
		return matches, len(matches) > 0
	}

	fullPath := filepath.Join(soFar, dir, repo)

	// check if the directory is a repository
	dirIsRepo := !checkIsRepo || isRepo(fullPath)

	if repo == dir && dirIsRepo {
		mtx.Lock()
		matches[fullPath] = struct{}{}
		mtx.Unlock()
		return matches, true
	}

	// handle partial name matches
	f, err := os.Stat(fullPath)
	if err == nil && f.IsDir() && dirIsRepo {
		mtx.Lock()
		matches[fullPath] = struct{}{}
		mtx.Unlock()
		return matches, len(matches) > 0
	}

	files, err := os.ReadDir(filepath.Join(soFar, dir))
	if err != nil {
		return matches, len(matches) > 0
	}

	// collect results in a thread-local manner
	localMatches := make(map[string]struct{})
	var localMtx sync.Mutex
	wg := sync.WaitGroup{}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		wg.Add(1)
		go func(file fs.DirEntry) {
			defer wg.Done()
			newMatches, ok := in(repo, file.Name(), filepath.Join(soFar, dir), make(map[string]struct{}), checkIsRepo, depth+1, mtx)
			if ok {
				localMtx.Lock()
				for match := range newMatches {
					localMatches[match] = struct{}{}
				}
				localMtx.Unlock()
			}
		}(file)
	}
	wg.Wait()

	// merge local matches into global matches
	mtx.Lock()
	for match := range localMatches {
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

	// Regex to parse different Git URL formats
	patterns := []string{
		`^git@([^:]+):(.+?)(?:\.git)?$`,                         // SSH: git@host:path
		`^(?:https?|ssh)://(?:[^@]+@)?([^/]+)/(.+?)(?:\.git)?$`, // HTTP/HTTPS/SSH with protocol
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(url); matches != nil {
			host := matches[1]
			path := matches[2]
			return fmt.Sprintf("https://%s/%s", host, path)
		}
	}

	// Fallback: assume it's already in a clean format
	return "https://" + strings.TrimSuffix(url, ".git")
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
