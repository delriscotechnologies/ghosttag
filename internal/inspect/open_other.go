//go:build !linux

package inspect

import (
	"fmt"
	"os"
)

func openForInspection(path string) (*os.File, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	after, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !after.Mode().IsRegular() || !os.SameFile(before, after) {
		_ = file.Close()
		return nil, fmt.Errorf("file changed while it was being opened or is not a regular file")
	}

	return file, nil
}
