package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
	Entries []dict.Entry
}

type ReaderHistItem struct {
	Sha256 string
	Name   string
	Data   []byte
}
type ReaderHist struct {
	Hist       [10]ReaderHistItem
	Start, Len int
	mtx        sync.RWMutex
}

func loadHistFromFile() {
	data, err := os.ReadFile(histFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("WARN: could not read:", histFile, "reason:", err)
		}
		return
	}

	if err = json.Unmarshal(data, &readerHist); err != nil {
		fmt.Println("WARN: could not parse json:", err)
		return
	}
	fmt.Println("INFO: loaded history form:", histFile)
}

func (rh *ReaderHist) saveToFile() {
	data, err := json.Marshal(rh)
	if err != nil {
		fmt.Println("WARN: could not make json:", err)
		return
	}

	f, err := os.Create(histFile)
	if err != nil {
		fmt.Println("WARN: could not create:", histFile, "reason:", err)
		return
	}
	defer f.Close()
	if _, err = f.Write(data); err != nil {
		fmt.Println("WARN: could not write to:", histFile, "reason:", err)
		return
	}
}

var (
	readerHist ReaderHist
	histFile   = func() string {
		const n = ".dict_history.json"
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, n)
		}
		return n
	}()
)

func (s *server) readerHandler(w http.ResponseWriter, r *http.Request) {
	t := s.t
	if debug {
		t = p(template.ParseGlob("pub/*.html"))
	}

	txt := strings.TrimSpace(r.FormValue("txt"))
	if txt == "" {
		readerHist.mtx.RLock()
		defer readerHist.mtx.RUnlock()

		sha := strings.TrimSpace(r.FormValue("hist"))
		if sha == "" {
			var s strings.Builder
			for i := 0; i < readerHist.Len; i++ {
				idx := readerHist.Start + i
				idx %= len(readerHist.Hist)
				a := fmt.Sprintf(`<a class="hist-item" href="/rd?hist=%s">- %s</a>`,
					readerHist.Hist[idx].Sha256, html.EscapeString(readerHist.Hist[idx].Name))
				s.WriteString(a)
			}
			if err := t.ExecuteTemplate(w, "readerInpt.html",
				template.HTML(s.String())); debug && err != nil {
				panic(err)
			}
		} else {
			for i := 0; i < readerHist.Len; i++ {
				idx := readerHist.Start + i
				idx %= len(readerHist.Hist)
				if sha == readerHist.Hist[i].Sha256 {
					w.Write(readerHist.Hist[idx].Data)
					return
				}
			}
			http.Redirect(w, r, "/rd", http.StatusMovedPermanently)
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
		if f {
			if pageName == "" {
				pageName = l
			}
			f = !dict.ContainsArabic(l)
		}
		for _, w := range strings.Split(l, " ") {
			if w != "" {
				wr := s.d.FindWord(w)
				cp = append(cp, ReaderWord{w, wr})
			}
		}
		reader = append(reader, cp)
	}

	buf := new(bytes.Buffer)
	if err := t.ExecuteTemplate(buf, "reader.html", reader); debug && err != nil {
		panic(err)
	}
	w.Write(buf.Bytes())

	// thread safe code from here
	readerHist.mtx.Lock()
	defer readerHist.mtx.Unlock()

	sha := fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))
	for i := 0; i < len(readerHist.Hist); i++ {
		if sha == readerHist.Hist[i].Sha256 {
			return
		}
	}

	// wraping time. the hist is full..
	if readerHist.Len >= len(readerHist.Hist) {
		// the item wich was inserted first will be rewritten
		readerHist.Start = (readerHist.Start + 1) % len(readerHist.Hist)
		readerHist.Len--
	}
	idx := (readerHist.Start + readerHist.Len) % len(readerHist.Hist)
	readerHist.Hist[idx] = ReaderHistItem{sha, pageName, buf.Bytes()}
	readerHist.Len++
	readerHist.saveToFile()
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
