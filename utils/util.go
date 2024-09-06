package utils

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

var ErrInvalidAddr = errors.New("invalid address")

// ConvertAddr2Num convert ip address to number
// 127.0.0.1:6001 -> 2130706433, 6001
func ConvertAddr2Num(ipAddr string) (uint32, uint16, error) {
	// Split the IP address and port
	parts := strings.Split(ipAddr, ":")
	if len(parts) != 2 {
		return 0, 0, ErrInvalidAddr
	}
	// Parse the IP address
	ip := net.ParseIP(parts[0])
	if ip == nil {
		return 0, 0, ErrInvalidAddr
	}
	// Convert IP to uint32
	ip = ip.To4()
	if ip == nil {
		return 0, 0, ErrInvalidAddr
	}
	ipNum := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])

	// Parse the port
	port, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return 0, 0, ErrInvalidAddr
	}
	return ipNum, uint16(port), nil
}

// ConvertNum2Addr convert number to ip address
// 2130706433, 6001 -> 127.0.0.1:6001
func ConvertNum2Addr(ipNum uint32, port uint16) string {
	ip := net.IPv4(byte(ipNum>>24), byte(ipNum>>16), byte(ipNum>>8), byte(ipNum))
	return ip.String() + ":" + strconv.Itoa(int(port))
}
