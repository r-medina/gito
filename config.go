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

func (c *Config) Close() error {
	return c.f.Close()
}

type File interface {
	io.ReadWriteSeeker
	Sync() error
	Truncate(int64) error
	Close() error
}

func LoadConfig(workspace string) (*Config, error) {
	f, newConfig, err := initConfig()
	if err != nil {
		return nil, err
	}

	config := &Config{f: f}

	return config, loadConfig(f, config, newConfig, workspace)
}

func loadConfig(f File, config *Config, newConfig bool, workspace string) error {
	if newConfig {
		path := os.Getenv("GOPATH")
		if path == "" {
			var err error
			path, err = os.UserHomeDir()
			if err != nil {
				return err
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
			return err
		}

		return f.Sync()
	}

	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(config); err != nil {
		return err
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
			return fmt.Errorf("no workspace %q", workspace)
		}

		config.active = config.Workspaces[0]
	}

	return nil
}

func initConfig() (File, bool, error) {
	newConfig := false
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, newConfig, err
	}

	configDir := filepath.Join(home, "/.config/gito")
	configFile := filepath.Join(configDir, "gito.yaml")

	f, err := os.OpenFile(configFile, os.O_SYNC|os.O_RDWR, 0622)
	if os.IsNotExist(err) {
		newConfig = true
		err = os.MkdirAll(configDir, 0755)
		if err != nil {
			f.Close()
			return nil, newConfig, err
		}

		f, err = os.OpenFile(configFile, os.O_SYNC|os.O_WRONLY|os.O_CREATE, 0622)
		if err != nil {
			f.Close()
			return nil, newConfig, err

		}
	} else if err != nil {
		f.Close()
		return nil, newConfig, err
	}

	return f, false, nil
}
