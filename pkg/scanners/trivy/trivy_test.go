package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCommaSeparatedOptions(t *testing.T) {
	/*
	   The following tests will be run against this map:

	   m := map[string]bool{
	       "one": false,
	       "two": false,
	   }

	   This keys represent all allowable values and the value will be overwritten
	   if supplied by the user.
	*/
	inputs := []struct {
		options   string
		expectErr bool
		mapState  map[string]bool
	}{
		{
			options:   "three,four",
			expectErr: true,
		},
		{
			options:   "one,two",
			expectErr: false,
			mapState: map[string]bool{
				"one": true,
				"two": true,
			},
		},
		{
			options:   "two",
			expectErr: true,
			mapState: map[string]bool{
				"one": false,
				"two": true,
			},
		},
	}

	for i := range inputs {
		input := inputs[i]

		m := map[string]bool{
			"one": false,
			"two": false,
		}

		err := parseCommaSeparatedOptions(m, input.options)
		if !input.expectErr && err != nil {
			t.Error(err)
			continue
		}

		for k, v := range input.mapState {
			if m[k] != v {
				t.Errorf("expected '%s' to be '%t' but found '%t'\n", k, v, m[k])
			}
		}
	}
}

func TestDownloadAndInitDB(t *testing.T) {
	tmp, err := os.MkdirTemp(os.TempDir(), "eraser-trivy-scanner-test")
	defer os.RemoveAll(tmp)
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	cfg.CacheDir = tmp

	err = downloadAndInitDB(cfg)
	if err != nil {
		t.Fatal(err)
	}

	dbPaths := []string{
		filepath.Join(tmp, "db", "metadata.json"),
		filepath.Join(tmp, "db", "trivy.db"),
	}

	for _, path := range dbPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
}
