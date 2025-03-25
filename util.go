package main

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
)

func p[T any](r T, e error) T {
	if e != nil {
		panic(e)
	}
	return r
}

func findFreePort() string {
	const from, to = 8080, 8090
	for i := from; i <= to; i++ {
		p := strconv.Itoa(i)
		c, err := net.Listen("tcp", "0.0.0.0:"+p)
		if err == nil {
			err := c.Close()
			if err == nil {
				return p

			}
		}
	}

	fmt.Printf("findFreePort: count not find a free port! from %d to %d\n",
		from, to)
	os.Exit(1)
	return ""
}

func localIp() string {
	if runtime.GOOS == "windows" {
		return "localhost"
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}
	return "localhost"
}
