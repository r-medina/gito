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

	return g.find(repo, true)
}

func (g *G) find(repo string, checkIsRepo bool) ([]string, error) {
	matches := []string{}
	for _, dir := range g.config.active.path {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // ignore errors accessing files
			}

			if d.IsDir() && d.Name() == ".git" {
				return filepath.SkipDir
			}

			// Calculate depth relative to the root dir
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return nil
			}
			depth := strings.Count(rel, string(os.PathSeparator))
			if depth > 4 {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if !d.IsDir() {
				return nil
			}

			// Check if the directory name matches the repo we are looking for
			matched := d.Name() == repo
			if !matched && strings.Contains(repo, string(os.PathSeparator)) {
				if strings.HasSuffix(path, repo) {
					// Check boundary to avoid partial name match
					if len(path) == len(repo) || path[len(path)-len(repo)-1] == os.PathSeparator {
						matched = true
					}
				}
			}

			if matched {
				if !checkIsRepo || isRepo(path) {
					matches = append(matches, path)
				}
			}

			return nil
		})
		if err != nil {
			// NOTE: WalkDir error (shouldn't happen with our ignore policy,
			// but good to log/handle if we had a logger)
			return nil, err
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("%q not found", repo)
	}

	return matches, nil
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
	_, err := g.find(self, false)
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

	where, err := g.find(self, false)
	if err != nil {
		return "", err
	}

	return where[0], nil
}
