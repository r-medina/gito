package gito

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkComplexWhere creates a large, irregular directory structure
// to stress-test the Where function.
func BenchmarkComplexWhere(b *testing.B) {
	dir, err := ioutil.TempDir("", "gito-complex-bench-")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a complex structure:
	// - Depth up to 6
	// - Width up to 5
	// - ~5000 directories
	// - ~20000 files
	// - 1 target repo hidden somewhere deep
	target := createComplexStructure(b, dir, 4, 5, 0)

	// Ensure we actually created the target
	if target == "" {
		// Fallback: force create it at a known location
		target = filepath.Join(dir, "src", "github.com", "target-repo")
		err := os.MkdirAll(filepath.Join(target, ".git"), 0755)
		if err != nil {
			b.Fatal(err)
		}
	}
	// fmt.Printf("Target repo created at: %s\n", target)
	// fmt.Printf("Root dir: %s\n", dir)

	config := &Config{
		Workspaces: []*Workspace{{
			Name:    "default",
			Path:    dir,
			path:    []string{dir, filepath.Join(dir, "src")},
			Aliases: map[string]string{},
			Custom:  map[string]string{},
		}},
	}
	config.active = config.Workspaces[0]
	g := New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches, err := g.Where("target-repo")
		if err != nil {
			b.Fatal(err)
		}
		if len(matches) == 0 {
			b.Fatal("target repo not found")
		}
	}
}

func createComplexStructure(b *testing.B, root string, maxDepth, maxWidth, currentDepth int) string {
	if currentDepth >= maxDepth {
		return ""
	}

	// Randomize width for irregularity
	width := rand.Intn(maxWidth) + 1
	var targetPath string

	for i := 0; i < width; i++ {
		dirName := fmt.Sprintf("dir-%d-%d", currentDepth, i)
		dirPath := filepath.Join(root, dirName)

		// 1 in 1000 chance to be the target repo
		if rand.Intn(1000) == 0 && targetPath == "" {
			dirPath = filepath.Join(root, "target-repo")
			err := os.MkdirAll(filepath.Join(dirPath, ".git"), 0755)
			if err != nil {
				b.Fatal(err)
			}
			targetPath = dirPath
		} else {
			err := os.MkdirAll(dirPath, 0755)
			if err != nil {
				b.Fatal(err)
			}

			// Create some dummy files to slow down ReadDir
			for j := 0; j < 5; j++ {
				err := os.WriteFile(filepath.Join(dirPath, fmt.Sprintf("file-%d.txt", j)), []byte("content"), 0644)
				if err != nil {
					b.Fatal(err)
				}
			}

			// Recurse
			found := createComplexStructure(b, dirPath, maxDepth, maxWidth, currentDepth+1)
			if found != "" {
				targetPath = found
			}
		}
	}
	return targetPath
}
