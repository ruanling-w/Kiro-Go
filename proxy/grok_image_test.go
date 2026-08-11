package proxy

import (
	"context"
	"net/netip"
	"testing"
)

func TestPublicImageAddress(t *testing.T) {
	tests := []struct {
		address string
		public  bool
	}{
		{"8.8.8.8", true},
		{"2606:4700:4700::1111", true},
		{"127.0.0.1", false},
		{"10.0.0.1", false},
		{"169.254.169.254", false},
		{"100.64.0.1", false},
		{"192.0.2.1", false},
		{"::1", false},
		{"fc00::1", false},
		{"fe80::1", false},
		{"2001:db8::1", false},
		{"::ffff:127.0.0.1", false},
	}
	for _, test := range tests {
		t.Run(test.address, func(t *testing.T) {
			if got := publicImageAddress(netip.MustParseAddr(test.address).Unmap()); got != test.public {
				t.Fatalf("publicImageAddress(%s)=%v want %v", test.address, got, test.public)
			}
		})
	}
}

func TestFetchImageAsBase64RejectsUnsafeURLBeforeRequest(t *testing.T) {
	for _, rawURL := range []string{
		"http://example.com/image.png",
		"https://user:pass@example.com/image.png",
		"https://example.com:8443/image.png",
		"https://127.0.0.1/image.png",
		"https://[::1]/image.png",
	} {
		t.Run(rawURL, func(t *testing.T) {
			if _, err := fetchImageAsBase64(context.Background(), nil, rawURL); err == nil {
				t.Fatal("expected unsafe URL rejection")
			}
		})
	}
}
