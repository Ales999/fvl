package main

import (
	"fmt"
	"log"
	"os"
)

func main() {

	if len(os.Args) == 1 {
		fmt.Println("Example Usage:")
		fmt.Println(os.Args[0], "1.1.1.1 [2.2.2.2]")
		os.Exit(1)
	}

	var srcIp = os.Args[1]
	var dstIp string
	if len(os.Args) > 2 {
		dstIp = os.Args[2]
	}

	var scanFiles []string // Срез где будем хранить имена отобранных файлов для сканирования.

	dir, err := getCiscoConfigsPath("CISCONFS")
	if err != nil {
		fmt.Printf("Ошибка: %s.\n", err)
		os.Exit(1)
	}
	// Debug: 	fmt.Println("Путь для поиска:", dir)

	// Получить список элементов в директории
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	// Перебираем элементы в директории и отбираем только текстовые файлы.
	for _, e := range entries {
		// Если это директоря то пропускаем.
		if e.IsDir() {
			continue
		}
		var fName = e.Name()
		// Проверяем что это текстовый файл а не бинарный.
		fileStat, err := checkTextFile(&dir, &fName)
		if err != nil {
			continue
		}
		if fileStat {
			// Добавляем в список для сканирования.
			scanFiles = append(scanFiles, fName)
			//fmt.Println(fName)
		}
	}
	if len(scanFiles) == 0 {
		fmt.Println("Не найдены текстовые файлы для сканирования")
		os.Exit(1)
	}

	//fmt.Println(scanFiles)

	//ParseFiles(dir, scanFiles, "172.24.6.66", "172.24.64.194")
	ParseFiles(dir, scanFiles, srcIp, dstIp)

}
