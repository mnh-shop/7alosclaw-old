// Command zalo-landlock applies Landlock filesystem confinement to itself and
// then execs the given command.
//
// It is built into the 7alos sandbox image and confines tool execution to the
// tenant workspace as defense-in-depth on top of the container's namespaces,
// read-only root, cap-drop ALL, no-new-privileges and seccomp profile.
//
// Usage:
//
//	zalo-landlock [--rw <dir>]... [--ro <dir>]... [--strict] -- <cmd> [args...]
//
// Landlock is applied on a best-effort basis: if the running kernel lacks the
// requested ABI the closest supported version is used; if Landlock is entirely
// unsupported the command still runs (the container's other controls apply)
// unless --strict is passed, in which case it fails closed.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/landlock-lsm/go-landlock/landlock"
)

// readOnlySystemDirs are needed for a working shell and common runtimes.
var readOnlySystemDirs = []string{
	"/usr", "/bin", "/sbin", "/lib", "/lib64", "/etc", "/opt", "/proc", "/sys",
}

// writableRuntimeDirs back the container's tmpfs mounts and device nodes.
var writableRuntimeDirs = []string{"/tmp", "/var/tmp", "/run", "/dev"}

func main() {
	rwDirs, roDirs, strict, argv := parse(os.Args[1:])
	if len(argv) == 0 {
		fmt.Fprintln(os.Stderr, "zalo-landlock: no command given")
		os.Exit(2)
	}

	ro := append(append([]string{}, readOnlySystemDirs...), roDirs...)
	rw := append(append([]string{}, writableRuntimeDirs...), rwDirs...)

	rules := []landlock.Rule{
		landlock.RODirs(ro...).IgnoreIfMissing(),
		landlock.RWDirs(rw...).IgnoreIfMissing(),
	}

	if strict {
		if err := landlock.V3.RestrictPaths(rules...); err != nil {
			fail(fmt.Errorf("landlock (strict): %w", err))
		}
	} else if err := landlock.V3.BestEffort().RestrictPaths(rules...); err != nil {
		fmt.Fprintf(os.Stderr, "zalo-landlock: warning: %v (continuing)\n", err)
	}

	path, err := exec.LookPath(argv[0])
	if err != nil {
		fail(fmt.Errorf("look up %q: %w", argv[0], err))
	}
	if err := syscall.Exec(path, argv, os.Environ()); err != nil {
		fail(fmt.Errorf("exec %q: %w", argv[0], err))
	}
}

// parse splits the args into rw dirs, ro dirs, the strict flag and the command.
func parse(args []string) (rw, ro []string, strict bool, argv []string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--":
			return rw, ro, strict, args[i+1:]
		case "--rw":
			if i+1 < len(args) {
				rw = append(rw, args[i+1])
				i++
			}
		case "--ro":
			if i+1 < len(args) {
				ro = append(ro, args[i+1])
				i++
			}
		case "--strict":
			strict = true
		default:
			// First non-flag token starts the command.
			return rw, ro, strict, args[i:]
		}
	}
	return rw, ro, strict, nil
}

func fail(err error) {
	if errors.Is(err, syscall.ENOSYS) {
		fmt.Fprintln(os.Stderr, "zalo-landlock: Landlock not supported by this kernel")
	}
	fmt.Fprintln(os.Stderr, "zalo-landlock:", err)
	os.Exit(126)
}
