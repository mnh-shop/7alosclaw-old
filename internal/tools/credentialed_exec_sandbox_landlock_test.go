package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mnh-shop/7alosclaw/internal/sandbox"
)

// landlockRecordingSandbox captures the exact argv, workdir and exec options
// handed to the sandbox manager so the credentialed path can be asserted end
// to end (including env delivery on the same Exec call).
type landlockRecordingSandbox struct {
	command []string
	workDir string
	opts    sandbox.ExecOpts
}

func (s *landlockRecordingSandbox) Exec(_ context.Context, command []string, workDir string, opts ...sandbox.ExecOption) (*sandbox.ExecResult, error) {
	s.command = append([]string(nil), command...)
	s.workDir = workDir
	s.opts = sandbox.ApplyExecOpts(opts)
	return &sandbox.ExecResult{Stdout: "ok"}, nil
}

func (s *landlockRecordingSandbox) Destroy(context.Context) error { return nil }
func (s *landlockRecordingSandbox) ID() string                    { return "landlock-recording-sandbox" }

type landlockRecordingManager struct {
	sandbox *landlockRecordingSandbox
}

func (m *landlockRecordingManager) Get(context.Context, string, string, *sandbox.Config, ...sandbox.GetOption) (sandbox.Sandbox, error) {
	if m.sandbox == nil {
		m.sandbox = &landlockRecordingSandbox{}
	}
	return m.sandbox, nil
}

func (*landlockRecordingManager) Release(context.Context, string) error { return nil }
func (*landlockRecordingManager) ReleaseAll(context.Context) error      { return nil }
func (*landlockRecordingManager) Stop()                                 {}
func (*landlockRecordingManager) Stats() map[string]any                 { return nil }

// TestCredentialedSandboxExecWrapsBinaryInLandlock pins the credentialed
// sandbox path to the same Landlock launcher argv shape as the plain sandboxed
// path (shell.go executeInSandbox): `zalo-landlock --rw <cwd> -- <cmd> ...`.
// Credential env must ride the same Exec call so the launcher's syscall.Exec
// hands it to the wrapped child.
func TestCredentialedSandboxExecWrapsBinaryInLandlock(t *testing.T) {
	globalWorkspace := "/srv/zalosclaw/workspace"
	tenantWorkspace := "/srv/zalosclaw/workspace/tenants/acme"
	mgr := &landlockRecordingManager{}
	tool := NewSandboxedExecTool(globalWorkspace, true, mgr)

	result := tool.executeCredentialedSandbox(
		WithToolWorkspace(context.Background(), tenantWorkspace),
		"/usr/bin/gh",
		[]string{"api", "user"},
		tenantWorkspace,
		"session-1",
		map[string]string{"GH_TOKEN": "secret"},
		30*time.Second,
	)
	if result.IsError {
		t.Fatalf("executeCredentialedSandbox returned error: %s", result.ForLLM)
	}

	wantCommand := strings.Join(
		[]string{"zalo-landlock", "--rw", sandbox.DefaultContainerWorkdir, "--", "/usr/bin/gh", "api", "user"},
		" ",
	)
	if got := strings.Join(mgr.sandbox.command, " "); got != wantCommand {
		t.Fatalf("credentialed sandbox command = %q, want %q", got, wantCommand)
	}
	if mgr.sandbox.workDir != sandbox.DefaultContainerWorkdir {
		t.Fatalf("credentialed sandbox workDir = %q, want %q", mgr.sandbox.workDir, sandbox.DefaultContainerWorkdir)
	}
	if got := mgr.sandbox.opts.Env["GH_TOKEN"]; got != "secret" {
		t.Fatalf("credentialed sandbox env GH_TOKEN = %q, want secret", got)
	}
}
