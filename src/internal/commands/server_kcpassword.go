package commands

// kcpasswordCipher is the public XOR cipher macOS loginwindow uses to obfuscate
// /etc/kcpassword. It has been stable since Mac OS X 10.4 and is documented in
// numerous open-source auto-login tools (e.g. python-kcpassword). The bytes are
// not secret — they are just an obfuscation, not encryption — so storing them
// in source is fine.
var kcpasswordCipher = [...]byte{
	0x7D, 0x89, 0x52, 0x23, 0xD2, 0xBC, 0xDD, 0xEA, 0xA3, 0xB9, 0x1F,
}

// encodeKCPassword returns the bytes that go into /etc/kcpassword for the given
// account password. Format:
//   - XOR each password byte with cipher[i % len(cipher)]
//   - Pad with cipher bytes (continuing the cycle) up to the next multiple of 12,
//     and at least 12 bytes total — this ensures loginwindow can detect end-of-string
//     when the password length is exactly a multiple of 12.
func encodeKCPassword(password string) []byte {
	pw := []byte(password)
	out := make([]byte, len(pw), len(pw)+12)
	for i, b := range pw {
		out[i] = b ^ kcpasswordCipher[i%len(kcpasswordCipher)]
	}
	target := ((len(out) / 12) + 1) * 12
	for len(out) < target {
		out = append(out, kcpasswordCipher[len(out)%len(kcpasswordCipher)])
	}
	return out
}
