package main

import "net"

// Тестируем текстовый IP что это именно IP.
func checkIPAddress(ip string) bool {
	return net.ParseIP(ip) != nil
}
