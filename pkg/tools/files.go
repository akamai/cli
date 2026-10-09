package tools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// RemoveOutput removes an existing output file or symlink without failing when it is absent.
func RemoveOutput(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CreateExclusive removes any existing output and creates name exclusively in dir.
func CreateExclusive(dir, name string, perm os.FileMode) (*os.File, error) {
	if name == "" || name == "." || filepath.IsAbs(name) || filepath.Base(name) != name {
		return nil, fmt.Errorf("output name %q must be a file name", name)
	}

	dirInfo, err := os.Lstat(dir)
	if err != nil {
		return nil, fmt.Errorf("inspect output directory %q: %w", dir, err)
	}
	if !dirInfo.IsDir() {
		return nil, fmt.Errorf("output directory %q must be a non-symlink directory", dir)
	}

	outputPath := filepath.Join(dir, name)
	if err := RemoveOutput(outputPath); err != nil {
		return nil, fmt.Errorf("remove existing output %q: %w", outputPath, err)
	}

	file, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return nil, err
	}
	if runtime.GOOS != "windows" {
		if err := file.Chmod(perm); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("set output permissions %q: %w", outputPath, err)
		}
	}

	return file, nil
}

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
