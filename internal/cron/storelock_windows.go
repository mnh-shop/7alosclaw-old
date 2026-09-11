//go:build windows

package cron

import "os"

// lockStoreFile is a no-op on Windows (desktop edition, single instance).
func lockStoreFile(f *os.File) error { return nil }

// unlockStoreFile is a no-op on Windows.
func unlockStoreFile(f *os.File) {}
