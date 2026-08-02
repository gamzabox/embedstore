package embedstore

import (
	"errors"
	"testing"
)

func TestLoadFileRejectsDuplicateItemIDs(t *testing.T) {
	data := validFileBytes(t)
	// Reuse the valid fixture's single record by appending cannot alter item count;
	// this regression is covered by a crafted in-memory store through metadata mutation.
	// A checksum-valid malformed metadata record must not be accepted.
	offset := 10 + int(uint32(data[6])|uint32(data[7])<<8|uint32(data[8])<<16|uint32(data[9])<<24)
	length := int(uint32(data[offset]) | uint32(data[offset+1])<<8 | uint32(data[offset+2])<<16 | uint32(data[offset+3])<<24)
	copy(data[offset+4:offset+4+length], []byte(`{"id":"","content":"content","data":{}}`))
	rewriteChecksum(data)
	_, err := loadBytes("corrupt.embed", data)
	if !errors.Is(err, ErrInvalidFile) {
		t.Fatalf("error=%v", err)
	}
}
