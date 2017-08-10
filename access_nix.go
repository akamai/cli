// +build !windows

package main

import "golang.org/x/sys/unix"

const (
	ACCESS_R_OK = unix.R_OK
	ACCESS_W_OK = unix.W_OK
	ACCESS_X_OK = unix.X_OK
)

func checkAccess(path string, perm uint32) error {
	return unix.Access(path, perm)
}
