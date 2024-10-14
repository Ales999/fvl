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

type IpFullInfo struct {
	foundByIp  bool         // Совпадение найдено
	eualip     bool         // признак что искомый IP совпадает точно c найденным
	ifaceSatus bool         // Признак что интерфейс не выключен
	netPrefix  netip.Prefix // ip адрес и маска - пример: "192.168.1.1/24"
	hostname   string       // Имя хоста.
	vrfName    string       // Имя VRF
	faceName   string       // Имя интерфейса
	aclIn      string       // ACL на IN
	aclOut     string       // ACL на OUT
}

func NewIpFullInfo(
	foundByIp bool,
	eualip bool,
	ifaceSatus bool,
	hostname string,
	vrfName string,
	faceName string,
	netPrefix netip.Prefix,
	aclIn string,
	aclOut string,
) *IpFullInfo {
	return &IpFullInfo{
		foundByIp:  foundByIp,
		eualip:     eualip,
		ifaceSatus: ifaceSatus,
		hostname:   hostname,
		vrfName:    vrfName,
		faceName:   faceName,
		netPrefix:  netPrefix,
		aclIn:      aclIn,
		aclOut:     aclOut,
	}
}

// String - Перевести в строку данные структуры
func (inf *IpFullInfo) String() {

	if inf.eualip { // Если искомый ip точно совпадает - выделим цветом и префиксом
		fmt.Print("\u001b[31m!>\u001b[32m")
	}
	var statOff string
	// Если состояние интерфейса как административно выкдюченое (false) - то добавим инфомацию об этом.
	if !inf.ifaceSatus {
		statOff = " (DOWN)"
	}
	fmt.Print("Host: ", inf.hostname, " Iface: ", inf.faceName+statOff, " Vrf: ", inf.vrfName,
		" IfaceIp: ", inf.netPrefix.String(),
		" AclIn: ", inf.aclIn, " AclOut: ", inf.aclOut)

	if inf.eualip {
		fmt.Print("\u001b[0m\n")
	} else {
		fmt.Print("\n")
	}

}

type AgregInfo struct {
	src []IpFullInfo
	dst []IpFullInfo
}

func ParseFiles(patchForFiles string, fileNames []string, sourceIp string, destinationIp string) {

	var ainfo AgregInfo

	var sourceIpLen = len(sourceIp)
	var destinationIpLen = len(destinationIp)

	if sourceIpLen > 0 {
		// Перевоим из строки в netip.Adds
		srcAddr, err := netip.ParseAddr(sourceIp)
		if err != nil {
			fmt.Println("Error parsing", sourceIp, "Error: ", err)
			return
		}
		for _, file := range fileNames {
			parseFile := filepath.Join(patchForFiles, file)
			infs, err := ParseFile(parseFile, srcAddr)
			if err != nil {
				fmt.Println(err)
			}
			for _, inf := range infs {
				if inf.foundByIp {
					ainfo.src = append(ainfo.src, inf)
					//fmt.Println("Host:", inf.hostname, "Iface:", inf.faceName, "Vrf:", inf.vrfName, "AclIn:", inf.aclIn, "AclOut:", inf.aclOut)
				}
			}
		}
	}

	if destinationIpLen > 0 {
		// Перевоим из строки в netip.Adds
		dstAddr, err := netip.ParseAddr(destinationIp)
		if err != nil {
			fmt.Println("Error parsing", destinationIp, "Error: ", err)
			return
		}
		for _, file := range fileNames {
			parseFile := filepath.Join(patchForFiles, file)
			infs, err := ParseFile(parseFile, dstAddr)
			if err != nil {
				fmt.Println(err)
			}
			for _, inf := range infs {
				if inf.foundByIp {
					ainfo.dst = append(ainfo.dst, inf)
					//fmt.Println("Host:", inf.hostname, "Iface:", inf.faceName, "Vrf:", inf.vrfName, "AclIn:", inf.aclIn, "AclOut:", inf.aclOut)
				}
			}
		}
	}

	// Result:

	if sourceIpLen > 0 {
		//fmt.Println("Source:")
		for _, src := range ainfo.src {
			src.String()
		}
	}

	if destinationIpLen > 0 {
		fmt.Println("Destination:")
		for _, dst := range ainfo.dst {
			dst.String()
		}
	}

}

// Парсим файл.
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
	file.Close()

	var foundByIp bool
	var eualip bool            // Признак что IP совпадают
	var ifaceSatus bool = true // Признак что интерфейс не выключен административно (по дефолту - он рабочий)
	var hostname string        // Имя хоста.
	var hostNameFound bool     // Имя хоста в файле найдено или нет.
	var vrfName string         // Имя VRF
	var faceName string        // Имя интерфейса
	var onlyip netip.Addr      // только ip из найденной строки - пример "192.168.1.1"
	var netPrefix netip.Prefix // полностью ip адрес и маска - пример: "192.168.1.1/24"
	var aclIn string           // ACL на IN
	var aclOut string          // ACL на OUT

	for n, line := range txtlines {
		// Если имя хоста еще не нашли, то проверяем его.
		if !hostNameFound {
			if strings.HasPrefix(line, "hostname") {
				hostNameFound = true
				hostname = line[9:]
				//fmt.Println("HostName:", hostname)
				// Раз это была строка с именем хоста то дальше и проверть нет смысла.
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
						aclIn = ""
						aclOut = ""

						// Создаем новый срез без текущей строки и перебираем его для поиска ACL, если они есть.
						var bodyifaces = tlsts[f+1:]
						for _, body := range bodyifaces {
							// Если обнаружен конец блока (он больше не начинается с пробела) то прекращаем перебор
							if !strings.HasPrefix(body, " ") {
								break
							}
							// Проверим что интеофейс не выключен
							if strings.Contains(body, "shutdown") && !strings.Contains(body, "description") {
								ifaceSatus = false
							}

							if strings.HasPrefix(body, " ip access-group") {
								var aclName = parseAclName(body)
								if strings.HasSuffix(body, "in") {
									aclIn = aclName
								}
								if strings.HasSuffix(body, "out") {
									aclOut = aclName
								}
							}
						} // end for
					}
				}
			} // end for
		}
		if foundByIp {
			//fmt.Println(hostname, ifaceName, vrfName, prefix.String(), aclIn, aclOut)
			//ret = *NewIpFullInfo(foundByIp, eualip, ifaceSatus, hostname, vrfName, faceName, netPrefix, aclIn, aclOut)
			ret = append(ret, *NewIpFullInfo(foundByIp, eualip, ifaceSatus, hostname, vrfName, faceName, netPrefix, aclIn, aclOut))

		}
		foundByIp = false
		eualip = false
	}

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
func parseIpMaskFromLine(line string) (netip.Prefix, netip.Addr, error) {

	// Парсим строку - разложим ёё по частям
	cuttingByFour := strings.FieldsFunc(line, func(r rune) bool {
		return r == ' '
	})

	//ipStr := cuttingByFour[2]
	ipAddr, err := netip.ParseAddr(cuttingByFour[2])
	if err != nil {
		//fmt.Println("Error parsing IP from", cuttingByFour[2])
		return netip.Prefix{}, netip.Addr{}, err
	}

	parsedMask := cuttingByFour[3]
	stringMask := net.IPMask(net.ParseIP(parsedMask).To4())
	lengthMask, _ := stringMask.Size()

	var prefix = netip.PrefixFrom(ipAddr, lengthMask)

	//fmt.Println(ipAddr, " -:- ", maskStr, " Mask Leingt:", lengthMask, prefix.String())

	return prefix, ipAddr, nil

}
