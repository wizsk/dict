package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
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

type EntryInfo struct {
	Sha  string
	Name string
}

func writeEntieslist(w io.Writer, title, dir, extArg string) {
	if dir == "" {
		return
	}

	file, err := os.Open(filepath.Join(dir, entriesFileName))
	if err != nil {
		return
	}

	s := bufio.NewScanner(file)
	var files []EntryInfo

	for s.Scan() {
		b := bytes.SplitN(s.Bytes(), []byte{':'}, 2)
		if len(b) != 2 {
			log.Println("Warn: malformed data:", s.Text())
			continue
		}
		files = append(files, EntryInfo{
			Sha:  string(b[0]),
			Name: string(b[1]),
		})
	}
	if len(files) == 0 {
		return
	}

	fmt.Fprintln(w, title)
	for i := len(files) - 1; i >= 0; i-- {
		fmt.Fprintf(
			w,
			`<a class="hist-item" href="/rd/%s%s">- %s</a>`,
			files[i].Sha,
			extArg,
			html.EscapeString(files[i].Name))
	}
}
