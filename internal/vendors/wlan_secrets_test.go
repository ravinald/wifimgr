package vendors

import (
	"testing"

	"github.com/ravinald/wifimgr/internal/encryption"
)

func TestWLANHasPlaintextSecret(t *testing.T) {
	enc, err := encryption.Encrypt("p", "pw")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	cases := []struct {
		name string
		w    *WLAN
		want bool
	}{
		{"no secrets", &WLAN{SSID: "open"}, false},
		{"plaintext psk", &WLAN{PSK: "p"}, true},
		{"encrypted psk", &WLAN{PSK: enc}, false},
		{"plaintext radius", &WLAN{RadiusServers: []RadiusServer{{Host: "h", Secret: "s"}}}, true},
		{"encrypted radius", &WLAN{RadiusServers: []RadiusServer{{Host: "h", Secret: enc}}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := wlanHasPlaintextSecret(tc.w); got != tc.want {
				t.Errorf("wlanHasPlaintextSecret = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEncryptWLANSecrets(t *testing.T) {
	const pw = "battery-horse"
	w := &WLAN{
		PSK:           "top-secret",
		RadiusServers: []RadiusServer{{Host: "h", Secret: "radius-secret"}},
	}
	if err := encryptWLANSecrets(w, pw); err != nil {
		t.Fatalf("encryptWLANSecrets: %v", err)
	}

	if !encryption.IsEncrypted(w.PSK) {
		t.Fatalf("PSK not encrypted: %q", w.PSK)
	}
	if got, _ := encryption.Decrypt(w.PSK, pw); got != "top-secret" {
		t.Errorf("PSK round-trip = %q, want %q", got, "top-secret")
	}
	if !encryption.IsEncrypted(w.RadiusServers[0].Secret) {
		t.Fatalf("RADIUS secret not encrypted: %q", w.RadiusServers[0].Secret)
	}

	// Re-encrypting is a no-op: already-encrypted values pass through unchanged.
	prev := w.PSK
	if err := encryptWLANSecrets(w, pw); err != nil {
		t.Fatalf("second encryptWLANSecrets: %v", err)
	}
	if w.PSK != prev {
		t.Errorf("re-encrypt changed already-encrypted PSK")
	}
}
