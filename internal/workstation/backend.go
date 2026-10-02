// Package workstation defines the Backend/Session/Stream interfaces for remote
// execution environments. Phase 1 provides the registry and interfaces only —
// concrete implementations are added in Phase 2 (SSH) and Phase 3 (Docker).
package workstation

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/mnh-shop/7alosclaw/internal/store"
)

// Backend represents a connected remote execution environment.
// Implementations must be registered via Register() at init time.
type Backend interface {
	// Name returns the backend type identifier (e.g. "ssh" or "docker").
	Name() string
	// HealthCheck verifies the backend is reachable and operational.
	HealthCheck(ctx context.Context) error
	// OpenSession creates a new isolated execution session.
	OpenSession(ctx context.Context, sessionID string) (Session, error)
	// CloseSession terminates an open session by ID.
	CloseSession(ctx context.Context, sessionID string) error
	// Close shuts down the backend and releases all resources (connections, goroutines).
	Close() error
}

// ExecRequest describes a command to run in a Session.
type ExecRequest struct {
	Cmd        string
	Args       []string
	Env        map[string]string
	CWD        string
	Persistent bool          // if true, route via tmux (Phase 4)
	Timeout    time.Duration
}

// Session is a live connection to a workstation that can execute commands.
type Session interface {
	// ID returns the session identifier.
	ID() string
	// Exec runs a command and returns a Stream for I/O.
	Exec(ctx context.Context, req ExecRequest) (Stream, error)
	// Close terminates the session.
	Close(ctx context.Context) error
}

// Stream provides access to a running command's I/O and exit status.
type Stream interface {
	// Stdout returns the command's standard output reader.
	Stdout() io.Reader
	// Stderr returns the command's standard error reader.
	Stderr() io.Reader
	// Wait blocks until the command exits and returns its exit code.
	Wait() (exitCode int, err error)
	// Kill forcibly terminates the running command.
	Kill() error
}

// BackendFactory constructs a Backend from a registered Workstation record.
type BackendFactory func(ws *store.Workstation) (Backend, error)

// registry maps WorkstationBackend type → factory function.
// Populated by Phase 2+ init() calls via Register().
var registry = map[store.WorkstationBackend]BackendFactory{}

// Register adds a backend factory for the given backend type.
// Called from Phase 2 (ssh) and Phase 3 (docker) init() functions.
func Register(name store.WorkstationBackend, f BackendFactory) {
	registry[name] = f
}

// Open constructs a Backend for the given Workstation using the registered factory.
// Returns an error if no factory is registered for ws.BackendType.
func Open(ws *store.Workstation) (Backend, error) {
	f, ok := registry[ws.BackendType]
	if !ok {
		return nil, fmt.Errorf("backend not registered: %s", ws.BackendType)
	}
	return f(ws)
}

// ConnectionTestResult is what the workstations.testConnection RPC reports.
type ConnectionTestResult struct {
	Backend string `json:"backend"`
	// Runtime is the resolved container runtime ("podman"/"docker"); empty for
	// non-container backends.
	Runtime string `json:"runtime,omitempty"`
	// Image is the pinned image the backend will run; empty for non-container
	// backends.
	Image string `json:"image,omitempty"`
	// ImagePresent reports whether Image exists in the remote runtime's store.
	ImagePresent bool `json:"imagePresent"`
	// RootfulSocket is true when the resolved runtime is a rootful docker
	// daemon (host-root delegation even for a non-root uid).
	RootfulSocket bool `json:"rootfulSocket,omitempty"`
	// Message is an optional human-readable detail.
	Message string `json:"message,omitempty"`
}

// ConnectionTester is implemented by backends that can validate their
// connection AND report what they resolved (e.g. the container runtime and
// whether the pinned image is present). It is what the workstations.testConnection
// RPC calls. Kept separate from Backend so a backend that has nothing to resolve
// need not grow a method, and existing implementations/tests keep compiling.
type ConnectionTester interface {
	// TestConnection returns a sanitized, user-facing report. A non-nil result
	// may accompany a non-nil error (e.g. runtime resolved but image absent).
	TestConnection(ctx context.Context) (*ConnectionTestResult, error)
}

// TestConnection opens a transient Backend for ws, runs its TestConnection, and
// closes it. The workstations.testConnection RPC can probe a workstation without
// holding a BackendCache.
func TestConnection(ctx context.Context, ws *store.Workstation) (*ConnectionTestResult, error) {
	b, err := Open(ws)
	if err != nil {
		return nil, err
	}
	defer b.Close()
	tester, ok := b.(ConnectionTester)
	if !ok {
		return nil, fmt.Errorf("backend %q does not support connection testing", b.Name())
	}
	return tester.TestConnection(ctx)
}
