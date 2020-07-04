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
	Aliases map[string]string `yaml:"aliases,omitempty"`
	Custom  map[string]string `yaml:"custom,omitempty"`

	path []string
}

func (c *Config) sync() error {
	if err := c.f.Truncate(1); err != nil {
		return err
	}

	encoder := yaml.NewEncoder(c.f)
	if err := encoder.Encode(c); err != nil {
		return err
	}

	return c.f.Sync()
}

type File interface {
	io.ReadWriteSeeker // TODO
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

		ws := &Workspace{}
		if workspace == "" {
			ws.Name = "default"
		} else {
			ws.Name = workspace
		}
		ws.Path = path
		ws.path = filepath.SplitList(path)
		for i, path := range ws.path {
			ws.path[i] = filepath.Join(path, "src")
		}
		config.Workspaces = []*Workspace{ws}
		config.active = config.Workspaces[0]

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

	for _, w := range config.Workspaces {
		w.path = filepath.SplitList(w.Path)
		if w.Name == workspace {
			config.active = w
		}
	}

	if workspace == "" {
		config.active = config.Workspaces[0]
	}

	return config, nil
}
