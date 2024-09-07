package utils

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

// ConvertAddr2Num convert ip address to number
// 127.0.0.1:6001 -> 2130706433, 6001
func ConvertAddr2Num(ipAddr string) (uint32, uint16, error) {
	// Split the IP address and port
	//remove  " " in the ipAddr start and end
	ipAddr = strings.TrimSpace(ipAddr)
	parts := strings.Split(ipAddr, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("len(parts) != 2 %s", ipAddr)
	}
	// Parse the IP address
	ip := net.ParseIP(parts[0])
	if ip == nil {
		return 0, 0, fmt.Errorf("the ip is nil %s", parts[0])
	}
	// Convert IP to uint32
	ip = ip.To4()
	if ip == nil {
		return 0, 0, fmt.Errorf("ip.To4() is nil %s", parts[0])
	}
	ipNum := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])

	// Parse the port
	port, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return 0, 0, fmt.Errorf("strconv.ParseUint(parts[1], 10, 16) %s", parts[1])
	}
	return ipNum, uint16(port), nil
}

// ConvertNum2Addr convert number to ip address
// 2130706433, 6001 -> 127.0.0.1:6001
func ConvertNum2Addr(ipNum uint32, port uint16) string {
	ip := net.IPv4(byte(ipNum>>24), byte(ipNum>>16), byte(ipNum>>8), byte(ipNum))
	return ip.String() + ":" + strconv.Itoa(int(port))
}

func GenerateUUID() uint64 {
	// Generate a unique id using a random number generator
	return uint64(rand.Uint32())<<32 | uint64(rand.Uint32())
}

func GenerateRandomNumber() uint16 {
	return uint16(rand.Uint32())
}
