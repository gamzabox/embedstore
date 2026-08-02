package embedstore

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBytesRejectsDeclaredLengthLimits(t *testing.T) {
	for name, mutate := range map[string]func([]byte){
		"manifest": func(data []byte) { binary.LittleEndian.PutUint32(data[6:10], uint32(maxManifestBytes+1)) },
		"metadata": func(data []byte) {
			offset := 10 + int(binary.LittleEndian.Uint32(data[6:10]))
			binary.LittleEndian.PutUint32(data[offset:offset+4], uint32(maxItemMetadataBytes+1))
		},
	} {
		t.Run(name, func(t *testing.T) {
			data := validFileBytes(t)
			mutate(data)
			rewriteChecksum(data)
			_, err := loadBytes("corrupt.embed", data)
			if !errors.Is(err, ErrInvalidFile) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
func TestLoadFileRejectsOversizedPathBeforeRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "huge.embed")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(maxEmbedFileBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	for _, load := range []func(string) (Manifest, error){VerifyFile} {
		_, err := load(path)
		if !errors.Is(err, ErrInvalidFile) {
			t.Fatalf("verify error=%v", err)
		}
	}
	if _, err := LoadFile(path); !errors.Is(err, ErrInvalidFile) {
		t.Fatalf("load error=%v", err)
	}
}
