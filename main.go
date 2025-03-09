package main

import (
	"embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/wizsk/dict/dict"
)

const debug = true

//go:embed pub/*
var staticData embed.FS

type Data struct {
	Word    string
	Entries []dict.Entry
}

func main() {
	if debug {
		fmt.Println("---- running in debug mode ----")
	}
	dict := dict.MakeData()
	tmpl := p(template.ParseFS(staticData, "pub/*.html"))

	// word res
	http.HandleFunc("/wr", func(w http.ResponseWriter, r *http.Request) {
		if debug {
			tmpl = p(template.ParseGlob("pub/*.html"))
		}
		d := Data{Word: strings.TrimSpace(r.FormValue("w"))}
		d.Entries = dict.FindWords(d.Word)
		if err := tmpl.ExecuteTemplate(w, "res.html", &d); debug && err != nil {
			panic(err)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if debug {
			tmpl = p(template.ParseGlob("pub/*.html"))
		}
		d := Data{Word: strings.TrimSpace(r.FormValue("w"))}
		d.Entries = dict.FindWords(d.Word)
		if err := tmpl.Execute(w, &d); debug && err != nil {
			panic(err)
		}
	})


	http.Handle("/pub/", http.FileServerFS(staticData))

	p := ":" + findFreePort()
	fmt.Println("serving at: http://" + localIp() + p)
	panic(http.ListenAndServe(p, nil))
}

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
