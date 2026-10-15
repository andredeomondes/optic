package atvremote

import (
	"bufio"
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	PairingPort = 6467
	ControlPort = 6466

	stepTimeout = 10 * time.Second
)

// ErrCodeTypo means the code fails the local checksum: the user mistyped it.
// Nothing was sent to the TV, so the same session can retry.
var ErrCodeTypo = errors.New("código não confere (erro de digitação?)")

// Tracer receives every frame sent (">") or received ("<"). Useful for RE.
type Tracer func(dir string, msg []byte)

// Pairing is an in-progress Polo session. After Start returns, the TV is
// showing a 6-character hex code that must be passed to Finish.
type Pairing struct {
	conn   *tls.Conn
	r      *bufio.Reader
	client *x509.Certificate
	server *x509.Certificate
	trace  Tracer

	ServerName string // name the TV reported in PairingRequestAck
}

// StartPairing connects to host:6467 and runs the handshake until the TV
// displays the pairing code.
func StartPairing(ctx context.Context, host string, cert tls.Certificate, clientName string, trace Tracer) (*Pairing, error) {
	conn, err := dialTLS(ctx, host, PairingPort, cert)
	if err != nil {
		return nil, err
	}
	p := &Pairing{conn: conn, r: bufio.NewReader(conn), trace: trace}

	if p.client, err = x509.ParseCertificate(cert.Certificate[0]); err != nil {
		conn.Close()
		return nil, err
	}
	peers := conn.ConnectionState().PeerCertificates
	if len(peers) == 0 {
		conn.Close()
		return nil, errors.New("TV não apresentou certificado")
	}
	p.server = peers[0]

	steps := []struct {
		send   []byte
		expect protowire.Number
	}{
		{msgPairingRequest(clientName), fieldPairingRequestAck},
		{msgOptions(), fieldOptions},
		{msgConfiguration(), fieldConfigurationAck},
	}
	for _, s := range steps {
		reply, err := p.roundTrip(s.send, s.expect)
		if err != nil {
			conn.Close()
			return nil, err
		}
		if s.expect == fieldPairingRequestAck {
			if ack, err := parseFields(reply); err == nil {
				if name, ok := ack.bytes(1); ok {
					p.ServerName = string(name)
				}
			}
		}
	}
	return p, nil
}

// Finish sends the secret derived from the code shown on the TV.
func (p *Pairing) Finish(code string) error {
	secret, err := pairingSecret(p.client, p.server, code)
	if err != nil {
		return err
	}
	_, err = p.roundTrip(msgSecret(secret), fieldSecretAck)
	return err
}

func (p *Pairing) Close() error { return p.conn.Close() }

func (p *Pairing) roundTrip(msg []byte, expect protowire.Number) ([]byte, error) {
	p.conn.SetDeadline(time.Now().Add(stepTimeout))
	defer p.conn.SetDeadline(time.Time{})

	if p.trace != nil {
		p.trace(">", msg)
	}
	if err := writeFrame(p.conn, msg); err != nil {
		return nil, err
	}
	reply, err := readFrame(p.r)
	if err != nil {
		return nil, fmt.Errorf("lendo resposta da TV: %w", err)
	}
	if p.trace != nil {
		p.trace("<", reply)
	}

	f, err := parseFields(reply)
	if err != nil {
		return nil, err
	}
	if status, _ := f.uint(2); status != statusOK {
		text := poloStatusText[status]
		if text == "" {
			text = "desconhecido"
		}
		return nil, fmt.Errorf("TV respondeu status %d (%s)", status, text)
	}
	payload, ok := f.bytes(expect)
	if !ok {
		return nil, fmt.Errorf("TV respondeu mensagem inesperada (esperava campo %d)", expect)
	}
	return payload, nil
}

// pairingSecret computes SHA-256(client N, client e, server N, server e, code[1:]).
// The first byte of the code is a checksum: it must equal the first byte of
// the hash, which lets us catch typos before talking to the TV.
func pairingSecret(client, server *x509.Certificate, code string) ([]byte, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != codeSymbols {
		return nil, fmt.Errorf("código deve ter %d caracteres", codeSymbols)
	}
	raw, err := hex.DecodeString(code)
	if err != nil {
		return nil, errors.New("código deve ser hexadecimal (0-9, A-F)")
	}

	cpub, ok1 := client.PublicKey.(*rsa.PublicKey)
	spub, ok2 := server.PublicKey.(*rsa.PublicKey)
	if !ok1 || !ok2 {
		return nil, errors.New("certificados precisam ser RSA")
	}

	h := sha256.New()
	h.Write(cpub.N.Bytes())
	h.Write(big.NewInt(int64(cpub.E)).Bytes())
	h.Write(spub.N.Bytes())
	h.Write(big.NewInt(int64(spub.E)).Bytes())
	h.Write(raw[1:])
	sum := h.Sum(nil)

	if sum[0] != raw[0] {
		return nil, ErrCodeTypo
	}
	return sum, nil
}

// ProbeControl connects to the control port and reads the TV's first message.
// An unpaired client is rejected during the TLS handshake or right after it.
func ProbeControl(ctx context.Context, host string, cert tls.Certificate) ([]byte, error) {
	conn, err := dialTLS(ctx, host, ControlPort, cert)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(stepTimeout))
	return readFrame(bufio.NewReader(conn))
}

func dialTLS(ctx context.Context, host string, port int, cert tls.Certificate) (*tls.Conn, error) {
	d := tls.Dialer{
		NetDialer: &net.Dialer{Timeout: 5 * time.Second},
		Config:    tlsConfig(cert),
	}
	c, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprint(port)))
	if err != nil {
		return nil, err
	}
	return c.(*tls.Conn), nil
}
