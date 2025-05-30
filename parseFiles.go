package main

import (
	"bufio"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
)

// Парсим каталог с файлами и собираем информацию о IP адресах
func ParseFiles(patchForFiles string, fileNames []string, sourceIp string, destinationIp string) {
	var ainfo AgregInfo

	parseAndAppend := func(ip string, dest *[]IpFullInfo) {
		if len(ip) > 0 {
			parsedAddr, err := netip.ParseAddr(ip)
			if err != nil {
				fmt.Println("Error parsing", ip, "Error: ", err)
				return
			}
			for _, file := range fileNames {
				parseFile := filepath.Join(patchForFiles, file)
				infs, err := ParseFile(parseFile, parsedAddr)
				if err != nil {
					fmt.Println(err)
					continue
				}
				for _, inf := range infs {
					if inf.foundByIp {
						*dest = append(*dest, inf)
					}
				}
			}
		}
	}

	parseAndAppend(sourceIp, &ainfo.src)
	parseAndAppend(destinationIp, &ainfo.dst)

	printResults(ainfo)
}

func printResults(ainfo AgregInfo) {
	if len(ainfo.src) > 0 {
		// fmt.Println("Source:")
		for _, src := range ainfo.src {
			src.String()
		}
	}

	if len(ainfo.dst) > 0 {
		fmt.Println("Destination:")
		for _, dst := range ainfo.dst {
			dst.String()
		}
	}
}

// Парсим найденный файл.
func ParseFile(fullPatchFile string, findedIp netip.Addr) ([]IpFullInfo, error) {
	var ret []IpFullInfo

	file, err := os.OpenFile(fullPatchFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла: %s", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var txtlines []string
	for scanner.Scan() {
		txtlines = append(txtlines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %s", err)
	}

	var foundByIp bool
	var eualip bool              // Признак что IP совпадают
	var ifaceSatus bool = true   // Признак что интерфейс не выключен административно (по дефолту - он рабочий)
	var secondaryIp bool = false // Найденный IP это seconary IP
	var hostname string          // Имя хоста.
	var hostNameFound bool       // Имя хоста в файле найдено или нет.
	var vrfName string           // Имя VRF
	var faceName string          // Имя интерфейса
	var onlyip netip.Addr        // только ip из найденной строки - пример "192.168.1.1"
	var netPrefix netip.Prefix   // полностью ip адрес и маска - пример: "192.168.1.1/24"
	var aclIn string             // ACL на IN
	var aclOut string            // ACL на OUT

	// Пробегаем по всему файлу строчка за строчкой.
	for n, line := range txtlines {
		// Если имя хоста еще не нашли, то проверяем его.
		if !hostNameFound {
			if strings.HasPrefix(line, "hostname") {
				hostNameFound = true
				hostname = line[9:]
				continue
			}
		}
		// Ищем строку 'interface '
		if strings.HasPrefix(line, "interface ") {
			faceName = parseInterfaceName(line)

			// Выбираем остатки что еще не сканировали в отдельный слайс (только следующие 20 строк)
			var tlsts []string
			if len(txtlines[n+1:]) > 22 { // Если осталось в файле больше 22 строк то берем только 21 строку
				tlsts = txtlines[n+1 : n+20]
			} else {
				tlsts = txtlines[n+1:]
			}

			//Очистим от старых записей.
			vrfName = ""
			ifaceSatus = true
			secondaryIp = false

			// Ищем строки с IP и MASK
			for f, tlst := range tlsts {
				// Если блок интерфейса заканчивается то прерываем данный for
				if !strings.HasPrefix(tlst, " ") {
					break
				}
				if strings.HasPrefix(tlst, " vrf forwarding") || strings.HasPrefix(tlst, " ip vrf forwarding") {
					vrfName = parseVrfName(tlst)
				}

				// Если нашли запись об IP/MASK
				if strings.HasPrefix(tlst, " ip address ") {

					netPrefix, onlyip, err = parseIpMaskFromLine(tlst)
					if err != nil {
						continue
					}

					// Если есть! совпадение префикса с искомым, то ищем все остальное.
					if netPrefix.Contains(findedIp) {
						if findedIp.Compare(onlyip) == 0 {
							eualip = true
						}
						foundByIp = true
						secondaryIp = false
						aclIn = ""
						aclOut = ""

						// Проверка что это возможно seconadry
						if strings.Contains(tlst, `secondary`) {
							// Если это так, то установим признак
							secondaryIp = true
						}

						// Создаем новый срез без текущей строки и перебираем его для поиска ACL, если они есть.
						var bodyifaces = tlsts[f+1:]
						for _, body := range bodyifaces {
							// Если обнаружен конец блока (он больше не начинается с пробела) то прекращаем перебор
							if !strings.HasPrefix(body, " ") {
								break
							}
							// Проверим что интерфейс не выключен
							if strings.Contains(body, "shutdown") && !strings.Contains(body, "description") {
								ifaceSatus = false
							}

							if strings.HasPrefix(body, " ip access-group") {
								var aclName = parseAclName(body)
								if strings.HasSuffix(body, " in") {
									aclIn = aclName
								}
								if strings.HasSuffix(body, " out") {
									aclOut = aclName
								}
							}
						} // end for 'bodyifaces'
					}

				} // End if found needed ip
				if foundByIp {
					ret = append(ret, *NewIpFullInfo(foundByIp, eualip, ifaceSatus, secondaryIp, hostname, vrfName, faceName, netPrefix, aclIn, aclOut))
				}
				foundByIp = false
				eualip = false

			} // end for 'tlsts'
		}

		foundByIp = false
		eualip = false
	} // end for 'txtlines'

	return ret, nil
}

// parseVrfName - Разбираем строку и возвращаем название VRF
func parseVrfName(line string) string {

	// Парсим строку - разложим по частям
	cuttingByTree := strings.FieldsFunc(line, func(r rune) bool {
		return r == ' '
	})

	// Если новый формат
	if strings.HasPrefix(line, " vrf forwarding") {
		return cuttingByTree[2]
	}
	// Иначе старый 'ip vrf forwarding ...'
	return cuttingByTree[3]
}

// parseAclName - Разбираем строку и возвращаем название ACL
func parseAclName(line string) string {
	// Парсим строку - разложим по частям
	cuttingByTree := strings.FieldsFunc(line, func(r rune) bool {
		return r == ' '
	})
	return cuttingByTree[2]
}

// parseInterfaceName - Разбираем строку и возвращаем название интерфейса
func parseInterfaceName(line string) string {
	// Парсим строку - разложим по частям
	cuttingByTree := strings.FieldsFunc(line, func(r rune) bool {
		return r == ' '
	})
	return cuttingByTree[1]

}

// parseIpMaskFromLine - Разбираем строку и возвращаем её IP и Netmask
//
// Input:
// ' ip address 172.24.62.201 255.255.255.248'
//
// Output (by netip.Prefix.String()):
// '172.24.62.201/29'
//
// parseIpMaskFromLine - парсит строку вида ' ip address 172.24.62.201 255.255.255.248'
func parseIpMaskFromLine(line string) (netip.Prefix, netip.Addr, error) {

	// Разбиваем строку на части по пробелам
	cuttingByFour := strings.FieldsFunc(line, func(r rune) bool {
		return r == ' '
	})

	// Проверяем, что строка содержит достаточное количество полей
	if len(cuttingByFour) < 4 {
		return netip.Prefix{}, netip.Addr{}, fmt.Errorf("недостаточное количество полей в строке")
	}

	ipStr := cuttingByFour[2]
	ipAddr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return netip.Prefix{}, netip.Addr{}, fmt.Errorf("ошибка парсинга IP: %w", err)
	}

	parsedMask := cuttingByFour[3]
	stringMask := net.IPMask(net.ParseIP(parsedMask).To4())
	if stringMask == nil {
		return netip.Prefix{}, netip.Addr{}, fmt.Errorf("неверная маска подсети")
	}
	lengthMask, _ := stringMask.Size()

	var prefix = netip.PrefixFrom(ipAddr, lengthMask)

	return prefix, ipAddr, nil
}
