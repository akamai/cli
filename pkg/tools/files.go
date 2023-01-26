package tools

import (
	"fmt"
	"io"
	"os"
)

// MoveFile must copy+unlink the file because moving files is broken across filesystems
func MoveFile(src, dst string) (err error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(source *os.File) {
		e := source.Close()
		if e != nil {
			err = e
		}
	}(source)

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(destination *os.File) {
		e := destination.Close()
		if e != nil {
			err = e
		}
	}(destination)
	_, err = io.Copy(destination, source)

	if err != nil {
		return err
	}

	err = os.Chmod(dst, 0755)
	if err != nil {
		return err
	}

	err = os.Remove(src)
	return err
}
