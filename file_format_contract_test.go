package embedstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"os"
	"testing"
	"time"
)

func TestWriteFileV1SequentialLittleEndianLayout(t *testing.T) {
	t.Parallel()
	manifest := Manifest{
		FormatVersion:   1,
		DatasetName:     "golden",
		DatasetVersion:  "v1",
		Embedding:       "openai/test",
		Dimensions:      2,
		VectorType:      "float32",
		Normalized:      true,
		ItemCount:       1,
		CreatedAt:       time.Date(2026, 8, 2, 3, 4, 5, 0, time.UTC),
		ContentIncluded: true,
	}
	items := []Item{{ID: "item-1", Content: "hello", Data: []byte(`{"kind":"golden"}`)}}
	path := t.TempDir() + "/golden.embed"
	if err := WriteFile(path, manifest, items, []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data[:4]), "EMBD"; got != want {
		t.Fatalf("magic = %q, want %q", got, want)
	}
	if got, want := binary.LittleEndian.Uint16(data[4:6]), uint16(1); got != want {
		t.Fatalf("version = %d, want %d", got, want)
	}
	manifestLength := int(binary.LittleEndian.Uint32(data[6:10]))
	if manifestLength <= 0 || 10+manifestLength >= len(data)-32 {
		t.Fatalf("manifest length = %d is outside file", manifestLength)
	}
	var gotManifest Manifest
	if err := json.Unmarshal(data[10:10+manifestLength], &gotManifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if gotManifest != manifest {
		t.Fatalf("manifest = %#v, want %#v", gotManifest, manifest)
	}
	itemOffset := 10 + manifestLength
	itemLength := int(binary.LittleEndian.Uint32(data[itemOffset : itemOffset+4]))
	if itemLength <= 0 || itemOffset+4+itemLength+8 != len(data)-32 {
		t.Fatalf("item record does not precede exactly two float32 values")
	}
	var gotItem Item
	if err := json.Unmarshal(data[itemOffset+4:itemOffset+4+itemLength], &gotItem); err != nil {
		t.Fatalf("unmarshal item: %v", err)
	}
	if gotItem.ID != items[0].ID || gotItem.Content != items[0].Content || !bytes.Equal(gotItem.Data, items[0].Data) {
		t.Fatalf("item = %#v, want %#v", gotItem, items[0])
	}
	vectorOffset := itemOffset + 4 + itemLength
	if got, want := binary.LittleEndian.Uint32(data[vectorOffset:]), math.Float32bits(1); got != want {
		t.Fatalf("first vector bits = %#x, want %#x", got, want)
	}
	if got := binary.LittleEndian.Uint32(data[vectorOffset+4:]); got != 0 {
		t.Fatalf("second vector bits = %#x, want 0", got)
	}
	sum := sha256.Sum256(data[:len(data)-32])
	if !bytes.Equal(sum[:], data[len(data)-32:]) {
		t.Fatal("checksum does not cover the complete preceding layout")
	}
}

func TestLoadFileRejectsMalformedMetadataAndTrailingBytesWithValidChecksum(t *testing.T) {
	t.Parallel()
	for name, corrupt := range map[string]func([]byte){
		"metadata JSON": func(data []byte) {
			offset := 10 + int(binary.LittleEndian.Uint32(data[6:10]))
			data[offset+4] = '!'
		},
		"trailing byte": func(data []byte) {
			body := append([]byte(nil), data[:len(data)-32]...)
			body = append(body, 0)
			sum := sha256.Sum256(body)
			copy(data[:0], append(body, sum[:]...))
		},
	} {
		t.Run(name, func(t *testing.T) {
			data := validFileBytes(t)
			if name == "trailing byte" {
				body := append([]byte(nil), data[:len(data)-32]...)
				body = append(body, 0)
				sum := sha256.Sum256(body)
				data = append(body, sum[:]...)
			} else {
				corrupt(data)
				rewriteChecksum(data)
			}
			_, err := loadBytes("corrupt.embed", data)
			if !errors.Is(err, ErrInvalidFile) {
				t.Fatalf("error = %v, want ErrInvalidFile", err)
			}
		})
	}
}

func FuzzLoadBytes(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("EMBD\x01\x00\x00\x00\x00\x00"))
	f.Add([]byte("not an embed file"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = loadBytes("fuzz.embed", data)
	})
}
