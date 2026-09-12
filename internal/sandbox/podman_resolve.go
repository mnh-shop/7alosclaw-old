package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// resolveHostWorkspacePath maps a container-local path to its host-side
// equivalent for container-out-of-container (sibling) mounts (Docker or Podman).
// When ZalosClaw runs inside a container and spawns sandbox containers, the
// workspace path (e.g., /app/workspace) only exists inside the ZalosClaw
// container — the sandbox needs the corresponding host path or volume name
// to mount it correctly.
//
// If not running in a container (no /.dockerenv or /run/.containerenv), returns
// localPath as-is. When containerized, resolution failures are FATAL (return an
// error): binding a guessed container-local path would silently mount the wrong
// source into the sandbox, so we refuse rather than fail open.
func resolveHostWorkspacePath(ctx context.Context, localPath string) (string, error) {
	// Not in a container — local path is the host path. Rootless Podman
	// containers expose /run/.containerenv (not only Docker's /.dockerenv).
	if _, err := os.Stat("/.dockerenv"); err != nil {
		if _, err := os.Stat("/run/.containerenv"); err != nil {
			return localPath, nil
		}
	}

	containerID := detectContainerID()
	if containerID == "" {
		return "", fmt.Errorf("sandbox.resolve: cannot determine container ID — refusing to guess a host mount source")
	}

	// Short timeout to avoid blocking on a stuck container engine.
	inspectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(inspectCtx, sandboxBinary(), "inspect", "--format", "{{json .Mounts}}", containerID).Output()
	if err != nil {
		return "", fmt.Errorf("sandbox.resolve: runtime inspect failed for container %s: %w", containerID, err)
	}

	var mounts []struct {
		Type        string `json:"Type"`
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		Name        string `json:"Name"`
	}
	if err := json.Unmarshal(out, &mounts); err != nil {
		return "", fmt.Errorf("sandbox.resolve: failed to parse mount list: %w", err)
	}

	targetDir := filepath.Clean(localPath)

	// Resolve symlinks so mount matching uses the canonical path. If the
	// resolved path doesn't match a known mount, the loop below fails closed.
	if resolved, err := filepath.EvalSymlinks(localPath); err == nil {
		resolvedClean := filepath.Clean(resolved)
		if resolvedClean != targetDir {
			slog.Debug("sandbox.resolve: canonicalized local mount path")
		}
		targetDir = resolvedClean
	}

	var bestDest string
	var bestSource string
	var bestRel string
	var bestType string
	var bestName string

	for _, m := range mounts {
		dest := filepath.Clean(m.Destination)
		if targetDir == dest || strings.HasPrefix(targetDir, dest+string(filepath.Separator)) {
			if len(dest) > len(bestDest) {
				bestDest = dest
				bestSource = m.Source
				bestType = m.Type
				bestName = m.Name
				bestRel, _ = filepath.Rel(dest, targetDir)
			}
		}
	}

	if bestDest == "" {
		return "", fmt.Errorf("sandbox.resolve: no matching mount for %s in container %s", localPath, containerID)
	}

	// Named volume: return volume name if it's the exact mount, otherwise
	// use the host source path (assumes local volume driver).
	if bestType == "volume" && bestName != "" {
		if bestRel == "." {
			slog.Debug("sandbox.resolve: resolved mount to named volume")
			return bestName, nil
		}
		if bestSource != "" {
			return filepath.Join(bestSource, bestRel), nil
		}
	}

	// Bind mount: join source with relative path.
	if bestSource != "" {
		resolved := filepath.Join(bestSource, bestRel)
		slog.Debug("sandbox.resolve: resolved mount to host path")
		return resolved, nil
	}

	return "", fmt.Errorf("sandbox.resolve: mount %s found but has no source path", bestDest)
}

// detectContainerID returns the current container ID using multiple
// strategies for reliability:
//  1. /proc/self/mountinfo — parse the container ID from cgroup mount paths
//  2. HOSTNAME env var — usually the short container ID (but can be overridden)
//  3. os.Hostname() — fallback
func detectContainerID() string {
	// Strategy 1: Parse /proc/self/mountinfo for docker container ID.
	// Lines contain paths like /docker/containers/<id>/...
	if data, err := os.ReadFile("/proc/self/mountinfo"); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			if _, after, ok := strings.Cut(line, "/docker/containers/"); ok {
				rest := after
				if slashIdx := strings.IndexByte(rest, '/'); slashIdx > 0 {
					id := rest[:slashIdx]
					if len(id) >= 12 { // Docker IDs are 64 hex chars, short form 12
						return id
					}
				}
			}
		}
	}

	// Strategy 2: HOSTNAME env var (Docker and Podman both set it to the short ID).
	if h := os.Getenv("HOSTNAME"); h != "" {
		return h
	}

	// Strategy 3: os.Hostname() fallback.
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}

	return ""
}
