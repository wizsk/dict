package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/wizsk/dict/dict"
)

const debug = !true

//go:embed pub/*
var staticData embed.FS

type Data struct {
	Word    string
	Entries []dict.Entry
}

type server struct {
	d dict.Dictionary
	t *template.Template
}

type ReaderWord struct {
	Word    string
	NoRes   bool
	Entries []dict.Entry
}

func (s *server) readerHandler(w http.ResponseWriter, r *http.Request) {
	t := s.t
	if debug {
		t = p(template.ParseGlob("pub/*.html"))
	}

	txt := strings.TrimSpace(r.FormValue("txt"))
	if txt == "" {
		readerHistRWM.RLock()
		defer readerHistRWM.RUnlock()

		histIdx, err := strconv.Atoi(strings.TrimSpace(r.FormValue("hist")))
		if err != nil {
			t.ExecuteTemplate(w, "readerInpt.html", &readerHistArr)
		} else {
			if histIdx < 0 || len(readerHistArr) <= histIdx {
				http.Redirect(w, r, "/rd", http.StatusMovedPermanently)
			} else {
				w.Write(readerHistArr[histIdx].data)
			}
		}
		return
	}

	pageName := ""
	sc := bufio.NewScanner(strings.NewReader(txt))
	reader := [][]ReaderWord{}
	for f := true; sc.Scan(); {
		// current pera
		cp := []ReaderWord{}
		l := strings.TrimSpace(sc.Text())
		if l == "" {
			continue
		}
		// 1st line && found arabic line
		if f && dict.ContainsArabic(l) {
			f = false
			pageName = l
		}
		for _, w := range strings.Split(l, " ") {
			if w != "" {
				wr := s.d.FindWord(w)
				cp = append(cp, ReaderWord{w, len(wr) == 0, wr})
			}
		}
		reader = append(reader, cp)
	}

	buf := new(bytes.Buffer)
	if err := t.ExecuteTemplate(buf, "reader.html", reader); debug && err != nil {
		panic(err)
	}
	w.Write(buf.Bytes())

	readerHistRWM.Lock()
	defer readerHistRWM.Unlock()

	sha := sha256.Sum256(buf.Bytes())
	for i := 0; i < len(readerHistArr); i++ {
		if sha == readerHistArr[i].sha256 {
			return
		}
	}

	readerHistArr = append(readerHistArr,
		ReaderHist{sha, pageName, buf.Bytes()})
}

type ReaderHist struct {
	sha256 [32]byte
	Name   string
	data   []byte
}

var (
	readerHistRWM sync.RWMutex
	readerHistArr []ReaderHist
)

func main() {
	if debug {
		fmt.Println("---- running in debug mode ----")
	}
	dict := dict.MakeData()
	tmpl := p(template.ParseFS(staticData, "pub/*.html"))
	sv := server{dict, tmpl}

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
