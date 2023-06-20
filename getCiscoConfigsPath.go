package main

import (
	"fmt"
	"os"
)

func getCiscoConfigsPath(envname string) (string, error) {

	dir, exists := os.LookupEnv(envname)
	if !exists {
		return "", fmt.Errorf("переменная окружения %s не определена", envname)
	}

	stat, err := patchExists(dir)
	if err != nil {
		return "", err
	}
	if !stat {
		return "", fmt.Errorf("путь указанный в переменной окружения %s не найден", envname)
	}

	return dir, nil
}
