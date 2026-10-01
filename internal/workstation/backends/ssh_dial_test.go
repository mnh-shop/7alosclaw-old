package backends

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/mnh-shop/7alosclaw/internal/store"
	"golang.org/x/crypto/ssh"
)

// testSigner generates an in-memory ed25519 SSH signer. No network, no fixtures.
func testSigner(t *testing.T) ssh.Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	return s
}

// testHostCert builds and CA-signs a host certificate for principal.
func testHostCert(t *testing.T, ca ssh.Signer, principal string) *ssh.Certificate {
	t.Helper()
	cert := &ssh.Certificate{
		CertType:        ssh.HostCert,
		Key:             testSigner(t).PublicKey(),
		ValidPrincipals: []string{principal},
		ValidAfter:      uint64(time.Now().Add(-time.Minute).Unix()),
		ValidBefore:     uint64(time.Now().Add(time.Hour).Unix()),
	}
	if err := cert.SignCert(rand.Reader, ca); err != nil {
		t.Fatalf("sign host cert: %v", err)
	}
	return cert
}

// Empty fingerprint and no CA must now fail closed. Under the old behaviour this
// returned a callback that accepted any key.
func TestBuildHostKeyCallback_FailsClosedWithoutTrustAnchor(t *testing.T) {
	meta := &store.SSHMetadata{Host: "helper.example", Port: 22, User: "ops"}

	cb, err := buildHostKeyCallback(meta)
	if err == nil {
		t.Fatal("expected error with no trust anchor, got nil (accept-anything regression)")
	}
	if cb != nil {
		t.Fatal("expected nil callback on the fail-closed path")
	}
	if !strings.Contains(err.Error(), "knownHostsFingerprint") && !strings.Contains(err.Error(), "hostCaKey") {
		t.Fatalf("error should name the missing setting, got: %v", err)
	}
}

func TestBuildHostKeyCallback_InsecureOptInAccepts(t *testing.T) {
	meta := &store.SSHMetadata{Host: "helper.example", AllowInsecureHostKey: true}

	cb, err := buildHostKeyCallback(meta)
	if err != nil {
		t.Fatalf("explicit opt-in should not error: %v", err)
	}
	if err := cb("helper.example:22", nil, testSigner(t).PublicKey()); err != nil {
		t.Fatalf("explicit opt-in should accept: %v", err)
	}
}

func TestBuildHostKeyCallback_FingerprintPin(t *testing.T) {
	host := testSigner(t)
	meta := &store.SSHMetadata{
		Host:                  "helper.example",
		KnownHostsFingerprint: ssh.FingerprintSHA256(host.PublicKey()),
	}

	cb, err := buildHostKeyCallback(meta)
	if err != nil {
		t.Fatalf("fingerprint pin should not error: %v", err)
	}
	if err := cb("helper.example:22", nil, host.PublicKey()); err != nil {
		t.Fatalf("matching pin should accept: %v", err)
	}
	if err := cb("helper.example:22", nil, testSigner(t).PublicKey()); err == nil {
		t.Fatal("mismatched key must be rejected")
	}
}

func TestBuildHostKeyCallback_TrustedCAHostCertAccepted(t *testing.T) {
	ca := testSigner(t)
	cert := testHostCert(t, ca, "helper.example")
	meta := &store.SSHMetadata{
		Host:      "helper.example",
		HostCAKey: string(ssh.MarshalAuthorizedKey(ca.PublicKey())),
	}

	cb, err := buildHostKeyCallback(meta)
	if err != nil {
		t.Fatalf("CA config should not error: %v", err)
	}
	if err := cb("helper.example:22", nil, cert); err != nil {
		t.Fatalf("cert from trusted CA must be accepted: %v", err)
	}
}

func TestBuildHostKeyCallback_UntrustedCAHostCertRejected(t *testing.T) {
	trustedCA := testSigner(t)
	rogueCA := testSigner(t)
	cert := testHostCert(t, rogueCA, "helper.example")
	meta := &store.SSHMetadata{
		Host:      "helper.example",
		HostCAKey: string(ssh.MarshalAuthorizedKey(trustedCA.PublicKey())),
	}

	cb, err := buildHostKeyCallback(meta)
	if err != nil {
		t.Fatalf("CA config should not error: %v", err)
	}
	if err := cb("helper.example:22", nil, cert); err == nil {
		t.Fatal("cert from untrusted CA must be rejected")
	}
}

// A CA configured but a plain (non-certificate) host key presented must be
// rejected — the CA is the only anchor.
func TestBuildHostKeyCallback_CAPlainKeyRejected(t *testing.T) {
	ca := testSigner(t)
	meta := &store.SSHMetadata{
		Host:      "helper.example",
		HostCAKey: string(ssh.MarshalAuthorizedKey(ca.PublicKey())),
	}

	cb, err := buildHostKeyCallback(meta)
	if err != nil {
		t.Fatalf("CA config should not error: %v", err)
	}
	if err := cb("helper.example:22", nil, testSigner(t).PublicKey()); err == nil {
		t.Fatal("plain host key must be rejected when only a CA anchor is configured")
	}
}
