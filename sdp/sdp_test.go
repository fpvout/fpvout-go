package sdp

import (
	"net"
	"testing"
)

func TestBuildVersionField(t *testing.T) {
	got := buildVersionField("1")
	want := "v=1\n"
	if got != want {
		t.Errorf("got %s want %s", got, want)
	}

}

func TestBuildOriginatorField(t *testing.T) {
	addr, _ := net.ResolveIPAddr("ip4", "1.2.3.4")
	tests := []struct {
		username  string
		startTime uint64
		endTime   uint64
		family    int
		address   *net.IPAddr
		want      string
	}{
		{"asdf", uint64(1234), uint64(5678), 7, addr, "o=asdf 1234 5678 IN IP7 1.2.3.4\n"},
	}
	for _, test := range tests {
		got := buildOriginatorField(test.username, test.startTime, test.endTime, test.family, *test.address)
		if got != test.want {
			t.Errorf("got %s want %s", got, test.want)
		}
	}
}
