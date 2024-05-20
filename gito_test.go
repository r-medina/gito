package gito

import (
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
