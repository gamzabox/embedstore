package embedstore

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"testing"
	"time"
)

func TestLoadFileRejectsTruncatedAndChecksumCorruption(t *testing.T) {
	t.Parallel()
	data := validFileBytes(t)
	for name, corrupted := range map[string][]byte{
		"truncated": data[:len(data)-1],
		"checksum":  append([]byte(nil), data[:len(data)-32]...),
	} {
		t.Run(name, func(t *testing.T) {
			if name == "checksum" {
				corrupted[10] ^= 0xff
				corrupted = append(corrupted, data[len(data)-32:]...)
			}
			_, err := loadBytes("corrupt.embed", corrupted)
			if !errors.Is(err, ErrInvalidFile) {
				t.Fatalf("error = %v, want ErrInvalidFile", err)
			}
			var fileError *FileError
			if !errors.As(err, &fileError) {
				t.Fatalf("error = %T, want FileError", err)
			}
		})
	}
}

func TestLoadFileRejectsNaNAndZeroVectorsWithValidChecksum(t *testing.T) {
	t.Parallel()
	for name, bits := range map[string]uint32{"nan": math.Float32bits(float32(math.NaN())), "zero": 0} {
		t.Run(name, func(t *testing.T) {
			data := validFileBytes(t)
			offset := firstVectorOffset(t, data)
			binary.LittleEndian.PutUint32(data[offset:], bits)
			if name == "zero" {
				binary.LittleEndian.PutUint32(data[offset+4:], 0)
			}
			rewriteChecksum(data)
			_, err := loadBytes("corrupt.embed", data)
			if !errors.Is(err, ErrInvalidFile) {
				t.Fatalf("error = %v, want ErrInvalidFile", err)
			}
		})
	}
}

func validFileBytes(t *testing.T) []byte {
	t.Helper()
	manifest := Manifest{FormatVersion: 1, DatasetName: "dataset", DatasetVersion: "v1", Embedding: "openai/model", Dimensions: 2, VectorType: "float32", Normalized: true, ItemCount: 1, CreatedAt: time.Now().UTC(), ContentIncluded: true}
	item := []Item{{ID: "item", Content: "content", Data: []byte(`{}`)}}
	path := t.TempDir() + "/fixture.embed"
	if err := WriteFile(path, manifest, item, []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func firstVectorOffset(t *testing.T, data []byte) int {
	t.Helper()
	offset := 10 + int(binary.LittleEndian.Uint32(data[6:10]))
	itemLength := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
	return offset + 4 + itemLength
}

func rewriteChecksum(data []byte) {
	sum := sha256.Sum256(data[:len(data)-32])
	copy(data[len(data)-32:], sum[:])
}
