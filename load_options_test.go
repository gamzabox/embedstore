package embedstore

import (
	"errors"
	"os"
	"testing"
)

func TestLoadFileChecksumOptionAndMemoryMode(t *testing.T) {
	path := t.TempDir() + "/fixture.embed"
	if err := os.WriteFile(path, validFileBytes(t), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] ^= 0xff
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); !errors.Is(err, ErrInvalidFile) {
		t.Fatalf("default error=%v", err)
	}
	verify := false
	if _, err := LoadFileWithOptions(path, LoadOptions{VerifyChecksum: &verify, MemoryMode: MemoryModeLoad}); err != nil {
		t.Fatalf("disabled checksum error=%v", err)
	}
	if _, err := LoadFileWithOptions(path, LoadOptions{MemoryMode: "mmap"}); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("mode error=%v", err)
	}
}

func TestLoadFileChecksumBypassStillValidatesStructure(t *testing.T) {
	path := t.TempDir() + "/fixture.embed"
	data := validFileBytes(t)
	data = data[:8]
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	disabled := false
	if _, err := LoadFileWithOptions(path, LoadOptions{VerifyChecksum: &disabled}); !errors.Is(err, ErrInvalidFile) {
		t.Fatalf("error=%v", err)
	}
}
