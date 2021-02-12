package gito

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Workspaces []*Workspace `yaml:"workspaces"`
	active     *Workspace
	f          File `yaml:"-"`
}

type Workspace struct {
	Name    string            `yaml:"name"`
	Path    string            `yaml:"path,omitempty"`
	Self    string            `yaml:"self,omitempty"`
	Aliases map[string]string `yaml:"aliases,omitempty"`
	Custom  map[string]string `yaml:"custom,omitempty"`

	path []string
}

func newWorkspace(name string) *Workspace {
	return &Workspace{
		Name:    name,
		Aliases: make(map[string]string),
		Custom:  make(map[string]string),
	}
}

func (ws *Workspace) Alias(alias string) (string, bool) {
	to, ok := ws.Aliases[alias]
	if !ok {
		return alias, false
	}

	return to, true
}

func (ws *Workspace) CustomPath(alias string) (string, bool) {
	to, ok := ws.Custom[alias]
	if !ok {
		return alias, false
	}

	return to, true
}

func (c *Config) Sync() error {
	if err := c.f.Truncate(0); err != nil {
		return err
	}

	if _, err := c.f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	encoder := yaml.NewEncoder(c.f)
	if err := encoder.Encode(c); err != nil {
		return err
	}

	return c.f.Sync()
}

type File interface {
	io.ReadWriteSeeker
	Sync() error
	Truncate(int64) error
}

func LoadConfig(f File, newConfig bool, workspace string) (*Config, error) {
	config := &Config{f: f}

	if newConfig {
		path := os.Getenv("GOPATH")
		if path == "" {
			var err error
			path, err = os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("making config file: %v", err)
			}
		}

		ws := newWorkspace(workspace)
		ws.Path = path
		ws.path = filepath.SplitList(path)
		for i, path := range ws.path {
			ws.path[i] = filepath.Join(path, "src")
		}
		config.Workspaces = []*Workspace{ws}
		config.active = config.Workspaces[0]

		if ws.Aliases == nil {
			ws.Aliases = make(map[string]string)
		}
		if ws.Custom == nil {
			ws.Custom = make(map[string]string)
		}

		encoder := yaml.NewEncoder(f)
		if err := encoder.Encode(config); err != nil {
			return nil, err
		}

		return config, f.Sync()
	}

	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("error decoding yaml config: %w", err)
	}

	if len(config.Workspaces) == 0 {
		w := newWorkspace(workspace)
		config.Workspaces = append(config.Workspaces, w)
		config.active = w
	}

	for _, w := range config.Workspaces {
		w.path = filepath.SplitList(w.Path)
		for i, path := range w.path {
			w.path[i] = filepath.Join(path, "src")
		}

		if w.Name == workspace {
			config.active = w
		}

		if w.Aliases == nil {
			w.Aliases = make(map[string]string)
		}
		if w.Custom == nil {
			w.Custom = make(map[string]string)
		}
	}

	if config.active == nil {
		if workspace != "" {
			return nil, fmt.Errorf("no workspace %q", workspace)
		}

		config.active = config.Workspaces[0]
	}

	return config, nil
}
