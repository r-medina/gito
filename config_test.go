package gito

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	r := strings.NewReader(`workspaces:
    personal:
      path: "~"
      aliases:
          g: gito
          d: dotfiles
      custom:
          dotfiles: "~/.dotfiles"
    work:
      path: "~/gh"
      overrideSrc: yes
      aliases:
          ghe: enterprise2`)

	config, err := loadConfig(r)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	t.Logf("CONFIG: %v", config)
}
