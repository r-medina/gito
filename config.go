package gito

import (
	"io"

	"gopkg.in/yaml.v3"
)

type config struct {
	Workspaces []struct {
		Name        string            `yaml:"name"`
		OverrideSrc bool              `yaml:overrideSrc,omitempty`
		Path        string            `yaml:"path,omitempty"`
		Aliases     map[string]string `yaml:"aliases,omitempty"`
		Custom      map[string]string `yaml:"custom,omitempty"`
	} `yaml:"workspaces"`
}

type Config struct {
	Workspaces map[string]struct {
		Name        string            `yaml:"name"`
		OverrideSrc bool              `yaml:overrideSrc,omitempty`
		Path        string            `yaml:"path,omitempty"`
		Aliases     map[string]string `yaml:"aliases,omitempty"`
		Custom      map[string]string `yaml:"custom,omitempty"`
	} `yaml:"workspaces"`
}

func loadConfig(r io.Reader) (*Config, error) {
	decoder := yaml.NewDecoder(r)
	config := &Config{}

	return config, decoder.Decode(config)
}
