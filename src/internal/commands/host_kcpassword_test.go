package commands

import (
	"bytes"
	"testing"
)

func TestEncodeKCPasswordEmpty(t *testing.T) {
	got := encodeKCPassword("")
	want := []byte{0x7D, 0x89, 0x52, 0x23, 0xD2, 0xBC, 0xDD, 0xEA, 0xA3, 0xB9, 0x1F, 0x7D}
	if !bytes.Equal(got, want) {
		t.Fatalf("empty password: got %x, want %x", got, want)
	}
}

func TestEncodeKCPasswordRoundtrip(t *testing.T) {
	cases := []string{"a", "hello", "12345678901", "12-character", "exactly-12-c", "this-is-a-longer-test-string-that-spans"}
	for _, pw := range cases {
		out := encodeKCPassword(pw)
		if len(out)%12 != 0 {
			t.Errorf("%q: output length %d not multiple of 12", pw, len(out))
		}
		if len(out) < 12 {
			t.Errorf("%q: output length %d < 12", pw, len(out))
		}
		// Decode by XORing with the same cipher and confirm the prefix matches.
		decoded := make([]byte, len(out))
		for i, b := range out {
			decoded[i] = b ^ kcpasswordCipher[i%len(kcpasswordCipher)]
		}
		if !bytes.HasPrefix(decoded, []byte(pw)) {
			t.Errorf("%q: decoded prefix mismatch: got %q", pw, string(decoded))
		}
	}
}
