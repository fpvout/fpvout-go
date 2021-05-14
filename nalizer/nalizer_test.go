package nalizer

import (
	"testing"
)

// TODO: Test frame counting

func TestNalizeShort(t *testing.T) {
	n := Nalizer{
		NALTypeLong: false,
	}
	nals := make([]NAL, 0)
	nals = append(nals, n.Nalize([]byte{0x99, 0x99})...)
	nals = append(nals, n.Nalize([]byte{0x00, 0x00, 0x01, 0x88})...)
	nals = append(nals, n.Nalize([]byte{0x88, 0x88, 0x88, 0x88})...)
	nals = append(nals, n.Nalize([]byte{0x00, 0x00, 0x00, 0x01})...)
	nals = append(nals, n.Nalize([]byte{0x12, 0x34, 0x56, 0x78})...)
	nals = append(nals, n.Nalize([]byte{0x12, 0x34, 0x00, 0x00})...)
	nals = append(nals, n.Nalize([]byte{0x00, 0x34, 0x00, 0x00})...)
	nals = append(nals, n.Nalize([]byte{0x01, 0x34, 0x00, 0x00})...)
	nals = append(nals, n.Nalize([]byte{0x00, 0x00, 0x00, 0x00})...)
	// Expect 2 NALs - end is not a full NAL, since we didn't get a new start
	expectedNals := 2
	if len(nals) != expectedNals {
		t.Errorf("got %d nals, want %d", len(nals), expectedNals)
	}
}

func TestNalizeLong(t *testing.T) {
	n := Nalizer{
		NALTypeLong: true,
	}
	nals := make([]NAL, 0)
	nals = append(nals, n.Nalize([]byte{0x12, 0x14})...)
	nals = append(nals, n.Nalize([]byte{0x03, 0x00, 0x00, 0x00})...)
	nals = append(nals, n.Nalize([]byte{0x01, 0x34, 0x56, 0x78})...)
	nals = append(nals, n.Nalize([]byte{0x12, 0x34, 0x00, 0x00})...)
	nals = append(nals, n.Nalize([]byte{0x01, 0x34, 0x00, 0x00})...)
	nals = append(nals, n.Nalize([]byte{0x01, 0x34, 0x00, 0x00})...)
	nals = append(nals, n.Nalize([]byte{0x00, 0x01, 0x00, 0x00})...)
	nals = append(nals, n.Nalize([]byte{0x12, 0x34, 0x00, 0x00})...)
	nals = append(nals, n.Nalize([]byte{0x09, 0x00, 0x00, 0x00})...)
	nals = append(nals, n.Nalize([]byte{0x01, 0x00, 0x00, 0x00})...)
	// Expect 2 NALs - end is not a full NAL, since we didn't get a new start
	expectedNals := 2
	if len(nals) != expectedNals {
		t.Errorf("got %d nals, want %d", len(nals), expectedNals)
	}
}
