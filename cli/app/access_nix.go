//go:build !windows

package app

import "golang.org/x/sys/unix"

func checkWriteAccess(path string) error {
	return unix.Access(path, unix.W_OK)
}
