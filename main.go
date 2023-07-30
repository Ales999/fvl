package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

var cli struct {
	SrcIp string `arg:"" required:"" name:"Src-IP" help:"Source IP"`
	DstIp string `arg:"" optional:"" name:"Dest-IP" help:"Destination IP"`
	// Flags:
	CfgDir string `required:"" help:"Path to backup cisco files" env:"CISCONFS" type:"existingdir"`
	Debug  bool   `help:"Enable more output"`
}

func main() {

	ctx := kong.Parse(&cli,
		kong.Name("fvl"),
		kong.Description("Find IP by VLAN"),
		kong.UsageOnError(),
	)

	if cli.Debug {
		log.Printf("Finded IP %s, Destination IP: %s", cli.SrcIp, cli.DstIp)
	}

	err := findByIPs(cli.SrcIp, cli.DstIp)
	ctx.FatalIfErrorf(err)
	os.Exit(0)

}

func findByIPs(srcIp string, dstIp string) error {

	// Срез где будем хранить имена отобранных файлов для сканирования.
	var scanFiles []string

	dir, err := getCiscoConfigsPath("CISCONFS")
	if err != nil {
		fmt.Printf("Ошибка: %s.\n", err)
		return err
	}

	if cli.Debug {
		log.Println("Путь для поиска:", dir)
	}

	// Получить список элементов в директории
	entries, err := os.ReadDir(dir)
	if err != nil {
		//log.Fatal(err)
		fmt.Printf("Ошибка: %s.\n", err)
		return err
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
			// Добавляем в список для сканирования только если это не '*.bak' или '*.backup' файл
			if checkNotBakFile(e) {
				scanFiles = append(scanFiles, fName)
				//fmt.Println(fName)
			}
		}
	}
	if len(scanFiles) == 0 {
		fmt.Println("Не найдены текстовые файлы для сканирования")
		os.Exit(1)
	}

	if cli.Debug {
		log.Println("Scan Files:", scanFiles)
		var logSt strings.Builder
		logSt.WriteString(fmt.Sprintf("SrcIP: %s ", srcIp))
		if len(dstIp) > 0 {
			logSt.WriteString(fmt.Sprintf("DstIP: %s ", dstIp))
		} else {
			logSt.WriteString("\n")
		}
		log.Println(logSt.String())
	}

	//ParseFiles(dir, scanFiles, "172.24.6.66", "172.24.64.194")
	ParseFiles(dir, scanFiles, srcIp, dstIp)

	return nil
}
