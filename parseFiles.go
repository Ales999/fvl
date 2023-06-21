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
	foundByIp bool         // Совпадение найдено
	hostname  string       // Имя хоста.
	vrfName   string       // Имя VRF
	faceName  string       // Имя интерфейса
	netPrefix netip.Prefix // ip адрес и маска - пример: "192.168.1.1/24"
	aclIn     string       // ACL на IN
	aclOut    string       // ACL на OUT
}

func NewIpFullInfo(
	foundByIp bool,
	hostname string,
	vrfName string,
	faceName string,
	netPrefix netip.Prefix,
	aclIn string,
	aclOut string,
) *IpFullInfo {
	return &IpFullInfo{
		foundByIp: foundByIp,
		hostname:  hostname,
		vrfName:   vrfName,
		faceName:  faceName,
		netPrefix: netPrefix,
		aclIn:     aclIn,
		aclOut:    aclOut,
	}
}

func (inf *IpFullInfo) String() {
	fmt.Println("Host:", inf.hostname, "Iface:", inf.faceName, "Vrf:", inf.vrfName,
		"IfaceIp:", inf.netPrefix.String(),
		"AclIn:", inf.aclIn, "AclOut:", inf.aclOut)
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
		for _, file := range fileNames {
			parseFile := filepath.Join(patchForFiles, file)
			srcAddr, err := netip.ParseAddr(sourceIp)
			if err != nil {
				fmt.Println("Error parsing", sourceIp, "Error: ", err)
			}
			inf, err := ParseFile(parseFile, srcAddr)
			if err != nil {
				fmt.Println(err)
			}
			if inf.foundByIp {
				ainfo.src = append(ainfo.src, inf)
				//fmt.Println("Host:", inf.hostname, "Iface:", inf.faceName, "Vrf:", inf.vrfName, "AclIn:", inf.aclIn, "AclOut:", inf.aclOut)
			}
		}
	}

	if destinationIpLen > 0 {
		for _, file := range fileNames {
			parseFile := filepath.Join(patchForFiles, file)
			dstAddr, err := netip.ParseAddr(destinationIp)
			if err != nil {
				fmt.Println("Error parsing", destinationIp, "Error: ", err)
			}
			inf, err := ParseFile(parseFile, dstAddr)
			if err != nil {
				fmt.Println(err)
			}
			if inf.foundByIp {
				ainfo.dst = append(ainfo.dst, inf)
				//fmt.Println("Host:", inf.hostname, "Iface:", inf.faceName, "Vrf:", inf.vrfName, "AclIn:", inf.aclIn, "AclOut:", inf.aclOut)
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
func ParseFile(fullPatchFile string, ip netip.Addr) (IpFullInfo, error) {

	var ret IpFullInfo

	file, err := os.OpenFile(fullPatchFile, os.O_RDONLY, 0644)
	if err != nil {
		return ret, fmt.Errorf("ошибка открытия файла: %s", err)
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
	var hostname string        // Имя хоста.
	var hostNameFound bool     // Имя хоста в файле найдено или нет.
	var vrfName string         // Имя VRF
	var faceName string        // Имя интерфейса
	var netPrefix netip.Prefix // ip адрес и маска - пример: "192.168.1.1/24"
	var aclIn string           // ACL на IN
	var aclOut string          // ACL на OUT

	for n, line := range txtlines {
		// Если имя хоста еще не нашли, то проверяем его.
		if !hostNameFound {
			if strings.HasPrefix(line, "hostname") {
				hostNameFound = true
				hostname = line[9:]
				//fmt.Println("HostName:", hostname)
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

					netPrefix, err = parseIpMaskFromLine(tlst)
					if err != nil {
						continue
					}

					// Если есть! совпадение префикса с искомым, то ищем все остальное.
					if netPrefix.Contains(ip) {
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
			ret = *NewIpFullInfo(foundByIp, hostname, vrfName, faceName, netPrefix, aclIn, aclOut)

		}
		foundByIp = false
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
func parseIpMaskFromLine(line string) (netip.Prefix, error) {

	// Парсим строку - разложим ёё по частям
	cuttingByFour := strings.FieldsFunc(line, func(r rune) bool {
		return r == ' '
	})

	//ipStr := cuttingByFour[2]
	ipAddr, err := netip.ParseAddr(cuttingByFour[2])
	if err != nil {
		//fmt.Println("Error parsing IP from", cuttingByFour[2])
		return netip.Prefix{}, err
	}

	parsedMask := cuttingByFour[3]
	stringMask := net.IPMask(net.ParseIP(parsedMask).To4())
	lengthMask, _ := stringMask.Size()

	var prefix = netip.PrefixFrom(ipAddr, lengthMask)

	//fmt.Println(ipAddr, " -:- ", maskStr, " Mask Leingt:", lengthMask, prefix.String())

	return prefix, nil

}
