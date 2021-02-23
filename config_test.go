package gito

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockConfigFile struct {
	io.Reader
}

func newMockConfigFile(s string) File {
	return &mockConfigFile{
		Reader: strings.NewReader(s),
	}
}

func (f *mockConfigFile) Write(p []byte) (n int, err error) {
	return 0, nil
}

func (f *mockConfigFile) Sync() error {
	return nil
}

func (f *mockConfigFile) Truncate(int64) error {
	return nil
}

func (f *mockConfigFile) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (f *mockConfigFile) Close() error {
	return nil
}

func TestLoad(t *testing.T) {
	r := `workspaces:
    - name: personal
      path: "/Users/ricky"
      aliases:
          g: gito
          d: dotfiles
      custom:
          dotfiles: "/Users/ricky/.dotfiles"
    - name: work
      path: "/Users/ricky/gh"
      aliases:
          ghe: super-secret
      custom:
          super-secret: "somewhereElse/theMoneyMaker"`

	f := newMockConfigFile(r)
	config := &Config{f: f}
	err := loadConfig(f, config, false, "")
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	got := config.Workspaces[0]
	want := &Workspace{
		Name:    "personal",
		Path:    "/Users/ricky",
		path:    []string{"/Users/ricky/src"},
		Aliases: map[string]string{"g": "gito", "d": "dotfiles"},
		Custom:  map[string]string{"dotfiles": "/Users/ricky/.dotfiles"},
	}

	assert := assert.New(t)
	assert.Equal(want, got)

	got = config.Workspaces[1]
	want = &Workspace{
		Name:    "work",
		Path:    "/Users/ricky/gh",
		path:    []string{"/Users/ricky/gh/src"},
		Aliases: map[string]string{"ghe": "super-secret"},
		Custom:  map[string]string{"super-secret": "somewhereElse/theMoneyMaker"},
	}
	assert.Equal(want, got)
}
