package embedstore

import (
	"errors"
	"os"
	"testing"
)

func TestChecksumOptionsKeepDefaultAndVerifyFileStrict(t *testing.T) {
	path := t.TempDir() + "/fixture.embed"
	data := validFileBytes(t)
	data[len(data)-1] ^= 0xff
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	var load func(string) (Store, error) = LoadFile
	for _, option := range []LoadOptions{{}, {VerifyChecksum: boolPointer(true)}} {
		_, err := LoadFileWithOptions(path, option)
		assertInvalidFileError(t, err)
	}
	_, err := load(path)
	assertInvalidFileError(t, err)
	disabled := false
	if _, err := LoadFileWithOptions(path, LoadOptions{VerifyChecksum: &disabled}); err != nil {
		t.Fatalf("disabled checksum=%v", err)
	}
	_, err = VerifyFile(path)
	assertInvalidFileError(t, err)
}
func boolPointer(value bool) *bool { return &value }
func assertInvalidFileError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrInvalidFile) {
		t.Fatalf("error=%v", err)
	}
	var fileError *FileError
	if !errors.As(err, &fileError) {
		t.Fatalf("missing FileError: %T", err)
	}
}
