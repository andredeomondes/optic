package atvremote

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"testing"
)

func testCert(t *testing.T) *x509.Certificate {
	t.Helper()
	tc, err := LoadOrCreateCert(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c, err := x509.ParseCertificate(tc.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// validCode builds the code a TV would display for this cert pair and nonce.
func validCode(client, server *x509.Certificate, nonce []byte) string {
	cpub := client.PublicKey.(*rsa.PublicKey)
	spub := server.PublicKey.(*rsa.PublicKey)
	h := sha256.New()
	h.Write(cpub.N.Bytes())
	h.Write(big.NewInt(int64(cpub.E)).Bytes())
	h.Write(spub.N.Bytes())
	h.Write(big.NewInt(int64(spub.E)).Bytes())
	h.Write(nonce)
	return strings.ToUpper(hex.EncodeToString(append([]byte{h.Sum(nil)[0]}, nonce...)))
}

func TestPairingSecret(t *testing.T) {
	client, server := testCert(t), testCert(t)
	code := validCode(client, server, []byte{0xBE, 0xEF})

	secret, err := pairingSecret(client, server, strings.ToLower(code))
	if err != nil {
		t.Fatalf("valid code rejected: %v", err)
	}
	if len(secret) != sha256.Size {
		t.Fatalf("secret len = %d", len(secret))
	}

	raw, _ := hex.DecodeString(code)
	raw[0] ^= 0xFF // corrupt the checksum byte
	bad := hex.EncodeToString(raw)
	if _, err := pairingSecret(client, server, bad); !errors.Is(err, ErrCodeTypo) {
		t.Fatalf("typo not detected, err = %v", err)
	}
	if _, err := pairingSecret(client, server, "12345"); err == nil {
		t.Fatal("short code accepted")
	}
	if _, err := pairingSecret(client, server, "ZZZZZZ"); err == nil {
		t.Fatal("non-hex code accepted")
	}
}

func TestFrameRoundTrip(t *testing.T) {
	msg := msgPairingRequest("OPTIC")
	f, err := parseFields(msg)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := f.uint(1); v != poloProtocolVersion {
		t.Errorf("protocol_version = %d", v)
	}
	if v, _ := f.uint(2); v != statusOK {
		t.Errorf("status = %d", v)
	}
	req, ok := f.bytes(fieldPairingRequest)
	if !ok {
		t.Fatal("missing pairing_request")
	}
	if !strings.Contains(DumpRaw(req), `"atvremote"`) {
		t.Errorf("dump missing service name:\n%s", DumpRaw(req))
	}
}
