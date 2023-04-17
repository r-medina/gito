package gito

import "testing"

func TestURL(t *testing.T) {
	urls := []string{
		"git@github.com:r-medina/gito.git",
		"https://github.com/r-medina/gito.git",
	}

	for _, url := range urls {
		urlStr, err := getURL(url)
		if err != nil {
			t.Errorf("error getting url for %q: %v", url, err)
		}

		if expected := "https://github.com/r-medina/gito.git"; urlStr != expected {
			t.Errorf("url for %q is %q, expected %q", url, expected, urlStr)
		}
	}
}
