//go:build integration

package gito

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAll contains integration tests for all the flows at the highest
// level.
func TestAll(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "gito-test-")
	assert.NoError(err, "making temp directory")
	defer os.RemoveAll(dir)

	f, err := os.Create(filepath.Join(dir, "config"))
	assert.NoError(err, "making config file")

	err = os.MkdirAll(filepath.Join(dir, "src"), 0755)
	assert.NoError(err, "making src dir")

	config := &Config{
		Workspaces: []*Workspace{{
			Name:    "default",
			Path:    dir,
			path:    []string{filepath.Join(dir, "src")},
			Aliases: map[string]string{},
			Custom:  map[string]string{},
		}},
		f: f,
	}
	config.active = config.Workspaces[0]

	g := New(config)

	//
	// test get
	//

	t.Run("get", func(t *testing.T) {
		err = g.Get("github.com/r-medina/gito")
		assert.NoError(err, "getting 'r-medina/gito'")

		// if this ever fails check to see if i've deleted the repo
		err = g.Get("https://github.com/r-medina/go.lang")
		assert.NoError(err, "getting 'https://github.com/r-medina/go.lang'")
	})

	//
	// test where
	//

	var where []string
	t.Run("get", func(t *testing.T) {
		// with full name of repo

		where, err = g.Where("github.com/r-medina/gito")
		assert.NoError(err, "where 'github.com/r-medina/gito")
		assert.Equal([]string{filepath.Join(dir, "src", "github.com/r-medina/gito")},
			where,
			"r-medina/gito not in expected location")

		// dropping github.com

		where, err = g.Where("r-medina/gito")
		assert.NoError(err, "where 'r-medina/gito")
		assert.Equal([]string{filepath.Join(dir, "src", "github.com/r-medina/gito")},
			where,
			"r-medina/gito not in expected location")

		// dropping r-medina

		where, err = g.Where("gito")
		assert.NoError(err, "where 'gito")
		assert.Equal([]string{filepath.Join(dir, "src", "github.com/r-medina/gito")},
			where,
			"gito not in expected location")

		// make sure get downloaded a repo

		assert.True(isRepo(where[0]), "%q is not a repo", where[0])

		// the one downloaded with https:// in front

		where, err = g.Where("go.lang")
		assert.NoError(err, "where 'github.com/r-medina/go.lang")
		assert.Equal([]string{filepath.Join(dir, "src", "github.com/r-medina/go.lang")},
			where,
			"r-medina/go.lang not in expected location")
	})

	//
	// test url
	//

	t.Run("url", func(t *testing.T) {
		url, err := g.URL("gito")
		assert.NoError(err, "getting 'gito' url")
		assert.Equal([]string{"https://github.com/r-medina/gito"}, url, "url for 'gito'")
	})

	//
	// test alias
	//

	t.Run("alias", func(t *testing.T) {
		err = g.Alias("g", "r-medina/gito")
		assert.NoError(err, "making alias g for r-medina/gito")

		alias, ok := config.active.Aliases["g"]
		assert.True(ok, "getting alias 'g' from workspace (using underlying map)")
		assert.Equal("r-medina/gito", alias, "alias value for 'g' (in underlying map)")
		alias, ok = config.active.Alias("g")
		assert.True(ok, "getting alias 'g' from workspace (using method Alias)")
		assert.Equal("r-medina/gito", alias, "alias value for 'g' (using method Alias)")

		// make sure where still works
		where, err = g.Where("g")
		assert.NoError(err, "where 'gito")
		assert.Equal([]string{filepath.Join(dir, "src", "github.com/r-medina/gito")},
			where,
			"'g' not in expected location")
	})

	//
	// test set
	//

	t.Run("set", func(t *testing.T) {
		want := filepath.Join(dir, "dotfiles")
		_, err = gitCloneAt("github.com/r-medina/interbtc", want)
		assert.NoError(err, "cloning dotfiles")

		err = g.Set("this", want)
		assert.NoError(err, "calling set")
		where, err = g.Where("this")
		assert.Equal([]string{want}, where, "calling where on 'this' after setting")
	})

	//
	// test self
	//

	t.Run("self", func(t *testing.T) {
		got, err := g.Self()
		assert.NoError(err, "getting self")
		assert.Equal("", got)
		assert.NoError(g.SetSelf("github.com/r-medina"))

		got, err = g.Self()
		assert.NoError(err, "getting self")
		assert.Equal(filepath.Join(dir, "src", "github.com/r-medina"), got)

	})
}
