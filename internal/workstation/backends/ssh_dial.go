package backends

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/mnh-shop/7alosclaw/internal/store"
	"golang.org/x/crypto/ssh"
)

// dialSSH establishes a new *ssh.Client using the provided metadata and key material.
// Context cancellation aborts the dial; the spawned goroutine cleans up on its own.
func dialSSH(ctx context.Context, meta *store.SSHMetadata, keyMaterial []byte) (*ssh.Client, error) {
	timeout := time.Duration(meta.ConnectTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	hostKeyCB, err := buildHostKeyCallback(meta)
	if err != nil {
		return nil, err
	}

	auth, err := buildAuthMethods(meta, keyMaterial)
	if err != nil {
		return nil, err
	}

	cfg := &ssh.ClientConfig{
		User:            meta.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCB,
		Timeout:         timeout,
	}

	addr := net.JoinHostPort(meta.Host, strconv.Itoa(meta.Port))

	type result struct {
		client *ssh.Client
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		c, e := ssh.Dial("tcp", addr, cfg)
		ch <- result{c, e}
	}()

	select {
	case r := <-ch:
		return r.client, r.err
	case <-ctx.Done():
		// Background goroutine will finish and its nascent connection will be discarded.
		return nil, ctx.Err()
	}
}

// buildHostKeyCallback returns an ssh.HostKeyCallback that verifies the host's
// identity against a configured trust anchor. Precedence:
//
//  1. HostCAKey — the host presents a certificate signed by a pinned CA
//     (ssh.CertChecker + IsHostAuthority). Survives host rebuilds: pin the CA
//     once, and each rebuilt host's fresh certificate verifies.
//  2. KnownHostsFingerprint — exact SHA256 pin of a single host key.
//  3. AllowInsecureHostKey — explicit dev-only opt-in to accept any key.
//
// If none of the above is configured the dial is refused before any network
// I/O, naming the missing setting (fail closed). An unpinned host is never
// accepted by default.
func buildHostKeyCallback(meta *store.SSHMetadata) (ssh.HostKeyCallback, error) {
	var caKey ssh.PublicKey
	if meta.HostCAKey != "" {
		k, _, _, _, err := ssh.ParseAuthorizedKey([]byte(meta.HostCAKey))
		if err != nil {
			return nil, fmt.Errorf("parse hostCaKey: %w", err)
		}
		caKey = k
	}
	hasPin := meta.KnownHostsFingerprint != ""

	if caKey == nil && !hasPin && !meta.AllowInsecureHostKey {
		return nil, fmt.Errorf(
			"ssh[%s]: refusing connection: no host trust anchor configured; "+
				"set knownHostsFingerprint to pin one host key, or hostCaKey to trust a signing CA "+
				"(allowInsecureHostKey exists but is for local development only)",
			meta.Host)
	}

	certChecker := &ssh.CertChecker{
		IsHostAuthority: func(auth ssh.PublicKey, _ string) bool {
			return caKey != nil && bytes.Equal(auth.Marshal(), caKey.Marshal())
		},
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fp := ssh.FingerprintSHA256(key)

		// 1. CA-signed host certificate (preferred; survives host rebuilds).
		if caKey != nil {
			if _, isCert := key.(*ssh.Certificate); isCert {
				if err := certChecker.CheckHostKey(hostname, remote, key); err == nil {
					return nil
				} else {
					slog.Warn("security.ssh_host_cert_rejected",
						"host", meta.Host,
						"fingerprint", fp,
						"err", err,
					)
				}
			}
		}

		// 2. Pinned single-host fingerprint.
		if hasPin {
			if fp != meta.KnownHostsFingerprint {
				slog.Warn("security.ssh_host_key_changed",
					"host", meta.Host,
					"expected", meta.KnownHostsFingerprint,
					"actual", fp,
				)
				return fmt.Errorf("host key mismatch for %s: expected %s got %s",
					meta.Host, meta.KnownHostsFingerprint, fp)
			}
			return nil
		}

		// 3. Explicit dev-only opt-in — loud by design.
		if meta.AllowInsecureHostKey {
			slog.Warn("security.ssh_host_key_insecure_accepted",
				"host", meta.Host,
				"fingerprint", fp,
				"hint", "allowInsecureHostKey is enabled — pin knownHostsFingerprint or hostCaKey for production",
			)
			return nil
		}

		// Fail closed (defence in depth; buildHostKeyCallback already rejected the
		// no-anchor case, but a CA-configured host presenting a plain key lands here).
		return fmt.Errorf("ssh[%s]: host key rejected: not signed by the configured CA and no fingerprint match", meta.Host)
	}, nil
}

// buildAuthMethods constructs SSH auth methods from metadata.
// Prefers public-key auth when keyMaterial is non-empty; falls back to password.
func buildAuthMethods(meta *store.SSHMetadata, keyMaterial []byte) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if len(keyMaterial) > 0 {
		signer, err := ssh.ParsePrivateKey(keyMaterial)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if meta.Password != "" {
		methods = append(methods, ssh.Password(meta.Password))
	}
	if len(methods) == 0 {
		return nil, errors.New("no auth method available: provide privateKey or password")
	}
	return methods, nil
}
