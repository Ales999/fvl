package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

var cli struct {
	SrcIp string `arg:"" required:"" name:"Src-IP" help:"Source IP"`
	DstIp string `arg:"" optional:"" name:"Dest-IP" help:"Destination IP"`
	// Flags:
	CfgDir string `required:"" help:"Path to backup cisco files" env:"CISCONFS" type:"existingdir"`
	Debug  bool   `help:"Enable more output" short:"d"`
}

func main() {

	ctx := kong.Parse(&cli,
		kong.Name("fvl"),
		kong.Description("Find VLAN by IP"),
		kong.UsageOnError(),
	)

	if cli.Debug {
		if len(cli.DstIp) > 0 {
			log.Printf("Finded Source IP: %s, Destination IP: %s\n", cli.SrcIp, cli.DstIp)
		} else {
			log.Printf("Finded IP: %s\n", cli.SrcIp)
		}
	}

	err := findByIPs()
	ctx.FatalIfErrorf(err)
	os.Exit(0)

}

func findByIPs() error {

	// Срез где будем хранить имена отобранных файлов для сканирования.
	var scanFiles []string
	// Директория где будем искать конфигурационные файлы.
	var dir = &cli.CfgDir

	// Уберем пробелы
	srcIp := strings.TrimSpace(cli.SrcIp)
	dstIp := strings.TrimSpace(cli.DstIp)

	// Check valied entered SRC IP
	if !checkIPAddress(srcIp) {
		return &net.ParseError{
			Type: "IP Source Address",
			Text: srcIp,
		}
	}

	// Check valied entered DST IP if entered
	if len(dstIp) > 0 {
		if !checkIPAddress(dstIp) {
			return &net.ParseError{
				Type: "IP Destination Address",
				Text: dstIp,
			}
		}
	}

	if cli.Debug {
		log.Println("Путь для поиска:", *dir)
	}

	// Получить список элементов в директории
	entries, err := os.ReadDir(*dir)
	if err != nil {
		fmt.Printf("Ошибка: %s.\n", err)
		return err
	}
	// Перебираем элементы в директории и отбираем только текстовые файлы.
	for _, entr := range entries {
		// Если это директоря то пропускаем.
		if entr.IsDir() {
			continue
		}
		var fName = entr.Name()
		// Проверяем что это текстовый файл а не бинарный.
		fileStat, err := checkTextFile(dir, &fName)
		if err != nil {
			continue
		}
		if fileStat {
			// Добавляем в список для сканирования только если это не '*.bak' или '*.backup' файл
			if checkNotBakFile(entr) {
				scanFiles = append(scanFiles, fName)
			}
		}
	}
	if len(scanFiles) == 0 {
		fmt.Println("Не найдены текстовые файлы бэкапов для сканирования")
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
	ParseFiles(*dir, scanFiles, srcIp, dstIp)

	return nil
}
