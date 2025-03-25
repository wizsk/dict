package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/wizsk/dict/dict"
)

const debug = !true

var (
	//go:embed pub/*
	staticData embed.FS
)

type Data struct {
	Word    string
	Entries []dict.Entry
}

type server struct {
	d dict.Dictionary
	t *template.Template
}

func main() {
	if debug {
		fmt.Println("---- running in debug mode ----")
	}
	dict := dict.MakeData()
	tmpl := p(template.ParseFS(staticData, "pub/*.html"))
	sv := server{dict, tmpl}
	loadHistFromFile()

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

	http.HandleFunc("/rd", sv.readerHandler)
	http.Handle("/pub/", http.FileServerFS(staticData))

	p := ":" + findFreePort()
	fmt.Println(os.Args[0] + ": serving at: http://" + localIp() + p)
	fmt.Println(os.Args[0] + ": serving at: http://localhost" + p)
	fmt.Println(os.Args[0] + ": reader at: http://" + localIp() + p + "/rd")
	fmt.Println(os.Args[0] + ": reader at: http://localhost" + p + "/rd")
	panic(http.ListenAndServe(p, nil))
}
