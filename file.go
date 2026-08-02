package embedstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
)

var fileMagic = [4]byte{'E', 'M', 'B', 'D'}

const fileVersion uint16 = 1

const (
	maxEmbedFileBytes    int64 = 1 << 30
	maxManifestBytes           = 16 << 20
	maxItemMetadataBytes       = 16 << 20
	maxItemCount               = 1_000_000
	maxVectorDimensions        = 1 << 20
	maxVectorValues            = 64 << 20
)

// WriteFile writes a complete v1 file. Vectors must be normalized and finite.
func WriteFile(path string, manifest Manifest, items []Item, vectors []float32) error {
	if manifest.FormatVersion == 0 {
		manifest.FormatVersion = int(fileVersion)
	}
	if manifest.VectorType == "" {
		manifest.VectorType = "float32"
	}
	if manifest.ItemCount == 0 {
		manifest.ItemCount = len(items)
	}
	store, err := NewMemoryStore(manifest, items, vectors)
	if err != nil {
		return err
	}
	manifest = store.manifest
	items = store.items
	vectors = store.vectors
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	var body bytes.Buffer
	body.Write(fileMagic[:])
	_ = binary.Write(&body, binary.LittleEndian, fileVersion)
	_ = binary.Write(&body, binary.LittleEndian, uint32(len(manifestJSON)))
	body.Write(manifestJSON)
	for _, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			return err
		}
		_ = binary.Write(&body, binary.LittleEndian, uint32(len(data)))
		body.Write(data)
	}
	for _, value := range vectors {
		_ = binary.Write(&body, binary.LittleEndian, value)
	}
	sum := sha256.Sum256(body.Bytes())
	body.Write(sum[:])
	return os.WriteFile(path, body.Bytes(), 0644)
}

// LoadFile verifies a file completely before returning an immutable memory store.
func LoadFile(path string) (Store, error) {
	data, err := readEmbedFile(path)
	if err != nil {
		return nil, err
	}
	return loadBytes(path, data)
}
func VerifyFile(path string) (Manifest, error) {
	data, err := readEmbedFile(path)
	if err != nil {
		return Manifest{}, err
	}
	s, err := loadBytes(path, data)
	if err != nil {
		return Manifest{}, err
	}
	return s.Manifest(), nil
}
func readEmbedFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() < 0 || info.Size() > maxEmbedFileBytes {
		return nil, &FileError{Path: path, Offset: 0, Err: ErrInvalidFile}
	}
	return os.ReadFile(path)
}

func loadBytes(path string, data []byte) (*MemoryStore, error) {
	fail := func(offset int, err error) (*MemoryStore, error) { return nil, &FileError{path, int64(offset), err} }
	if len(data) < 4+2+4+32 {
		return fail(0, ErrInvalidFile)
	}
	if !bytes.Equal(data[:4], fileMagic[:]) {
		return fail(0, ErrInvalidFile)
	}
	version := binary.LittleEndian.Uint16(data[4:6])
	if version != fileVersion {
		return fail(4, ErrUnsupportedVersion)
	}
	got := sha256.Sum256(data[:len(data)-32])
	if !bytes.Equal(got[:], data[len(data)-32:]) {
		return fail(len(data)-32, ErrInvalidFile)
	}
	off := 6
	n := int(binary.LittleEndian.Uint32(data[off : off+4]))
	off += 4
	if n < 2 || n > maxManifestBytes || n > len(data)-off-32 {
		return fail(off, ErrInvalidFile)
	}
	var m Manifest
	if err := json.Unmarshal(data[off:off+n], &m); err != nil {
		return fail(off, fmt.Errorf("%w: manifest: %v", ErrInvalidFile, err))
	}
	off += n
	if m.FormatVersion != int(fileVersion) || m.Dimensions <= 0 || m.Dimensions > maxVectorDimensions || m.ItemCount < 0 || m.ItemCount > maxItemCount || m.VectorType != "float32" || !m.Normalized {
		return fail(off, ErrInvalidFile)
	}
	items := make([]Item, m.ItemCount)
	for i := range items {
		if off+4 > len(data)-32 {
			return fail(off, ErrInvalidFile)
		}
		l := int(binary.LittleEndian.Uint32(data[off : off+4]))
		off += 4
		if l < 2 || l > maxItemMetadataBytes || l > len(data)-off-32 {
			return fail(off, ErrInvalidFile)
		}
		if err := json.Unmarshal(data[off:off+l], &items[i]); err != nil {
			return fail(off, fmt.Errorf("%w: metadata: %v", ErrInvalidFile, err))
		}
		if items[i].ID == "" || len(items[i].Data) == 0 {
			return fail(off, ErrInvalidFile)
		}
		off += l
	}
	if m.ItemCount != 0 && m.Dimensions > maxVectorValues/m.ItemCount {
		return fail(off, ErrInvalidFile)
	}
	count := m.ItemCount * m.Dimensions
	if count < 0 || count > (len(data)-off-32)/4 || off+count*4 != len(data)-32 {
		return fail(off, ErrInvalidFile)
	}
	vectors := make([]float32, count)
	for i := range vectors {
		vectors[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[off : off+4]))
		off += 4
	}
	for i := 0; i < m.ItemCount; i++ {
		var sq float64
		for _, x := range vectors[i*m.Dimensions : (i+1)*m.Dimensions] {
			if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
				return fail(off, ErrInvalidFile)
			}
			sq += float64(x) * float64(x)
		}
		if sq == 0 || math.Abs(math.Sqrt(sq)-1) > 1e-3 {
			return fail(off, ErrInvalidFile)
		}
	}
	return NewMemoryStore(m, items, vectors)
}

// InspectFile reads only the fixed header and manifest; it intentionally does not checksum vectors.
func InspectFile(path string) (Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return Manifest{}, err
	}
	defer f.Close()
	header := make([]byte, 10)
	if _, err = io.ReadFull(f, header); err != nil {
		return Manifest{}, &FileError{path, 0, ErrInvalidFile}
	}
	if !bytes.Equal(header[:4], fileMagic[:]) {
		return Manifest{}, &FileError{path, 0, ErrInvalidFile}
	}
	if binary.LittleEndian.Uint16(header[4:6]) != fileVersion {
		return Manifest{}, &FileError{path, 4, ErrUnsupportedVersion}
	}
	n := int(binary.LittleEndian.Uint32(header[6:10]))
	if n < 2 || n > 16<<20 {
		return Manifest{}, &FileError{path, 6, ErrInvalidFile}
	}
	raw := make([]byte, n)
	if _, err = io.ReadFull(f, raw); err != nil {
		return Manifest{}, &FileError{path, 10, ErrInvalidFile}
	}
	var m Manifest
	if err = json.Unmarshal(raw, &m); err != nil {
		return Manifest{}, &FileError{path, 10, ErrInvalidFile}
	}
	return m, nil
}
