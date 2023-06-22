package main

import (
	"io/fs"
	"path"
	"strings"
)

// Проверка имени файла на начинающиеся  с точки, а так-же с расширением *.bak и *.backup
func checkNotBakFile(f fs.DirEntry) bool {

	var finf, err = f.Info()
	if err != nil {
		return false
	}
	// Пропускаем файлы начинающиеся с точки ('.')
	if strings.HasPrefix(finf.Name(), ".") {
		return false
	}
	// Пропускаем файлы с расширеними 'bak' и 'backup'
	xtension := path.Ext(finf.Name())
	if strings.Compare(xtension, ".bak") == 0 || strings.Compare(xtension, ".backup") == 0 || strings.Compare(xtension, ".save") == 0 {
		return false
	}
	// Значит нам подходит, - будем добавлять в список для сканирования содержимого.
	return true

}
