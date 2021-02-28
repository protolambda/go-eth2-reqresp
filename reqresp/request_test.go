package reqresp

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestStreamHeaderAndPayloadSnappy(t *testing.T) {
	input, _ := hex.DecodeString("aabb1234")
	r := bytes.NewReader(input)
	var buf bytes.Buffer
	err := StreamHeaderAndPayload(uint64(len(input)), r, &buf, SnappyCompression{})
	if err != nil {
		t.Error(err)
	}
	expected, _ := hex.DecodeString("04ff060000734e6150705901080000e5310030aabb1234")
	if bytes.Compare(expected, buf.Bytes()) != 0 {
		t.Error("unexpected encoding output")
	}
}

func TestStreamHeaderAndPayload(t *testing.T) {
	input, _ := hex.DecodeString("aabb1234")
	r := bytes.NewReader(input)
	var buf bytes.Buffer
	err := StreamHeaderAndPayload(uint64(len(input)), r, &buf, nil) // no compression here
	if err != nil {
		t.Error(err)
	}
	expected, _ := hex.DecodeString("04aabb1234")
	if bytes.Compare(expected, buf.Bytes()) != 0 {
		t.Error("unexpected encoding output")
	}
}
