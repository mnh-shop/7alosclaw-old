//go:build !windows

package cron

import (
	"os"
	"syscall"
)

// lockStoreFile takes an exclusive non-blocking flock on the cron store file.
func lockStoreFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

// unlockStoreFile releases the flock.
func unlockStoreFile(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
