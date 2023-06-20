package main

import "net"

// Check if a certain ip in a cidr range.
//
// Examle use:
//
// test1, err := cidrRangeContains("10.0.0.0/24", "10.0.0.1") // true
//
// test2, err := cidrRangeContains("10.0.0.0/24", "127.0.0.1") // false
func cidrRangeContains(cidrRange string, checkIP string) (bool, error) {
	_, ipnet, err := net.ParseCIDR(cidrRange)
	if err != nil {
		return false, err
	}
	secondIP := net.ParseIP(checkIP)
	return ipnet.Contains(secondIP), err
}
