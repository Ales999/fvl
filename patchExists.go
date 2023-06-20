package main

import "os"

// exists returns whether the given file or directory exists
func patchExists(path string) (bool, error) {
	src, err := os.Stat(path)
	if err == nil {
		if src.IsDir() {
			return true, nil
		} else {
			return false, nil
		}
	}
	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}
