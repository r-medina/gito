package gito

import (
	"os"
	"path/filepath"
	"testing"
)

func TestURL(t *testing.T) {
	urls := []string{
		"git@github.com:r-medina/gito.git",
		"https://github.com/r-medina/gito.git",
	}

	for _, url := range urls {
		urlStr := extractURL(url)
		if expected := "https://github.com/r-medina/gito"; urlStr != expected {
			t.Errorf("url for %q is %q, expected %q", url, urlStr, expected)
		}
	}
}

func TestSelfNonRepo(t *testing.T) {
	// Create a temp dir structure
	dir, err := os.MkdirTemp("", "gito-self-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a directory that is NOT a git repo
	nonRepoDir := filepath.Join(dir, "src", "github.com", "my-non-repo")
	err = os.MkdirAll(nonRepoDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Workspaces: []*Workspace{{
			Name:    "default",
			Path:    dir,
			path:    []string{dir, filepath.Join(dir, "src")},
			Aliases: map[string]string{},
			Custom:  map[string]string{},
		}},
		f: &MockFile{},
	}
	config.active = config.Workspaces[0]
	g := New(config)

	// Try to set "my-non-repo" as Self
	// This should succeed if Self allows non-repos (old behavior)
	// This will fail if Self requires repos (new behavior)
	err = g.SetSelf("my-non-repo")
	if err != nil {
		t.Fatalf("SetSelf failed for non-repo: %v", err)
	}

	// Verify we can retrieve it
	got, err := g.Self()
	if err != nil {
		t.Fatalf("Self() failed: %v", err)
	}

	if got != nonRepoDir {
		t.Errorf("Self() = %v, want %v", got, nonRepoDir)
	}
}

func TestFindPathWithSlash(t *testing.T) {
	// Create a temp dir structure
	dir, err := os.MkdirTemp("", "gito-repro-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a directory structure: src/github.com/my-user
	targetDir := filepath.Join(dir, "src", "github.com", "my-user")
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Workspaces: []*Workspace{{
			Name:    "default",
			Path:    dir,
			path:    []string{filepath.Join(dir, "src")},
			Aliases: map[string]string{},
			Custom:  map[string]string{},
		}},
		f: &MockFile{},
	}
	config.active = config.Workspaces[0]
	g := New(config)

	// Try to find "github.com/my-user"
	// This mirrors the user's situation where Self is "github.com/r-medina"
	matches, err := g.find("github.com/my-user", false)
	if err != nil {
		t.Errorf("find failed: %v", err)
	} else if len(matches) != 1 || matches[0] != targetDir {
		t.Errorf("find returned wrong result: %v, want %v", matches, targetDir)
	}
}

type MockFile struct{}

func (m *MockFile) Read(p []byte) (n int, err error)             { return 0, nil }
func (m *MockFile) Write(p []byte) (n int, err error)            { return len(p), nil }
func (m *MockFile) Seek(offset int64, whence int) (int64, error) { return 0, nil }
func (m *MockFile) Sync() error                                  { return nil }
func (m *MockFile) Truncate(size int64) error                    { return nil }
func (m *MockFile) Close() error                                 { return nil }
