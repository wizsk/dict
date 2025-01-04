package main

import (
	"embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/wizsk/dict/dict"
)

const debug = !true

//go:embed index.html
var indexPage embed.FS

type Data struct {
	Word    string
	Entries []dict.Entry
}

func main() {
	if debug {
		fmt.Println("---- running in debug mode ----")
	}
	dict := dict.MakeData()
	tmpl := p(template.ParseFS(indexPage, "index.html"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if debug {
			tmpl = p(template.ParseFiles("index.html"))
		}
		d := Data{Word: strings.TrimSpace(r.FormValue("w"))}
		d.Entries = dict.FindWord(d.Word)
		if err := tmpl.Execute(w, &d); debug && err != nil {
			panic(err)
		}
	})

	p := ":" + findFreePort()
	fmt.Println("listening at: http://localhost" + p)
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
