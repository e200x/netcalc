package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	cliHelp = `Network Calculator v2.2
Usage: %[1]s <IP/CIDR>

Output:
  Address     - Input IP address with network mask
  Bitmask     - Number of bits in the network mask
  Netmask     - Subnet mask in dotted decimal format
  Wildcard    - Inverse mask for host calculations
  Network     - Base address of the network
  Broadcast   - Last address in the network
  Hostmin     - First usable host IP address
  Hostmax     - Last usable host IP address
  Hosts       - Total number of available hosts

Example:
  %[1]s 192.168.1.0/24
`
)

type ResultItem struct {
	Name  string
	Value string
}

func main() {
	flag.Usage = func() {
		fmt.Printf(cliHelp, os.Args[0])
	}
	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	runConsoleMode(flag.Arg(0))
}

func runConsoleMode(cidr string) {
	start := time.Now()
	ipStr, bitmaskStr, err := parseCIDR(cidr)
	if err != nil {
		log.Fatal("Error:", err)
	}

	result, err := calculate(ipStr, bitmaskStr)
	if err != nil {
		log.Fatal("Error:", err)
	}

	for _, item := range result {
		fmt.Printf("%-12s %s\n", item.Name+":", item.Value)
	}

	elapsed := time.Since(start)
	fmt.Printf("\nExecution time: %s\n", elapsed)
}

func parseCIDR(cidr string) (string, string, error) {
	if !strings.Contains(cidr, "/") {
		return "", "", fmt.Errorf("invalid CIDR format")
	}

	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", fmt.Errorf("invalid CIDR: %v", err)
	}

	parts := strings.Split(cidr, "/")
	return parts[0], parts[1], nil
}

func calculate(ipStr, bitmaskStr string) ([]ResultItem, error) {
	bitmask, err := strconv.Atoi(bitmaskStr)
	if err != nil || bitmask < 0 || bitmask > 32 {
		return nil, fmt.Errorf("invalid bitmask")
	}

	_, ipNet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", ipStr, bitmask))
	if err != nil {
		return nil, fmt.Errorf("invalid IP address")
	}

	if ipNet.IP.To4() == nil {
		return nil, fmt.Errorf("IPv6 is not supported")
	}

	ones, _ := ipNet.Mask.Size()
	netmask := net.IP(ipNet.Mask).To4()

	wildcard := make(net.IPMask, len(ipNet.Mask))
	for i := range ipNet.Mask {
		wildcard[i] = ^ipNet.Mask[i]
	}
	wildcardIP := net.IP(wildcard).To4()

	network := ipNet.IP.To4()

	broadcast := make(net.IP, len(network))
	copy(broadcast, network)
	for i := 0; i < 4; i++ {
		broadcast[i] |= ^ipNet.Mask[i]
	}

	var hostmin, hostmax net.IP
	switch {
	case ones == 32:
		hostmin = network
		hostmax = network
	case ones == 31:
		hostmin = network
		hostmax = broadcast
	default:
		hostmin = make(net.IP, len(network))
		copy(hostmin, network)
		incrementIP(hostmin)

		hostmax = make(net.IP, len(broadcast))
		copy(hostmax, broadcast)
		decrementIP(hostmax)
	}

	totalHosts := calculateTotalHosts(ones)

	return []ResultItem{
		{"Address", ipStr},
		{"Bitmask", fmt.Sprintf("%d", ones)},
		{"Netmask", netmask.String()},
		{"Wildcard", wildcardIP.String()},
		{"Network", network.String()},
		{"Broadcast", broadcast.String()},
		{"Hostmin", hostmin.String()},
		{"Hostmax", hostmax.String()},
		{"Hosts", fmt.Sprintf("%d", totalHosts)},
	}, nil
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

func decrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]--
		if ip[i] != 255 {
			break
		}
	}
}

func calculateTotalHosts(bitmask int) uint64 {
	if bitmask >= 31 {
		return uint64(32 - bitmask + 1)
	}
	return uint64(1<<(32-bitmask)) - 2
}