package embedstore

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

func TestLoadFileRejectsNonNormalizedVector(t *testing.T) {
	data := validFileBytes(t)
	offset := firstVectorOffset(t, data)
	binary.LittleEndian.PutUint32(data[offset:], math.Float32bits(2))
	binary.LittleEndian.PutUint32(data[offset+4:], math.Float32bits(0))
	rewriteChecksum(data)
	_, err := loadBytes("corrupt.embed", data)
	if !errors.Is(err, ErrInvalidFile) {
		t.Fatalf("error = %v, want ErrInvalidFile", err)
	}
}
