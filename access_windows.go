package main

const (
	ACCESS_R_OK = iota
	ACCESS_W_OK
	ACCESS_X_OK
)

func checkAccess(path string, perm int) error {
	return nil
}
