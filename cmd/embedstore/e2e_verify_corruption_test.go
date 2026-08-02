package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestVerifyCommandRejectsHeaderCorruption(t *testing.T) {
	for _, kind := range []string{"magic", "version", "truncated"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path := writeE2EEmbed(t, dir)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "magic":
				data[0] = 'X'
			case "version":
				data[4] = 2
			case "truncated":
				data = data[:8]
			}
			if err := os.WriteFile(filepath.Clean(path), data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := exec.Command(buildCLI(t, dir), "verify", "--file", path).Run(); err == nil {
				t.Fatal("expected non-zero exit")
			}
		})
	}
}
