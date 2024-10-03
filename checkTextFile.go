package main

import (
	"bufio"
	"os"
	"path/filepath"
	"unicode/utf8"
)

// Проверка что файл текстовый
func checkTextFile(origPath, fileNameCheck *string) (bool, error) {

	fullPath := filepath.Join(*origPath, *fileNameCheck)

	readFile, err := os.Open(fullPath)
	if err != nil {
		return false, err
	}
	defer readFile.Close()

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	fileScanner.Scan()

	return utf8.ValidString(string(fileScanner.Text())), nil
}
