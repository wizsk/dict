package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wizsk/dict/dict"
)

var (
	readerHist     ReaderHist
	readerHistFile = func() string {
		const n = ".dict_history.json"
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, n)
		}
		return n
	}()
)

type ReaderData struct {
	Name  string
	Peras [][]ReaderWord
}

type ReaderWord struct {
	AW      bool // Not arabic word
	Word    string
	Entries []dict.Entry
}

// this is used to store the history in file and mannge it.
// It actually saves the generated html page
type ReaderHist struct {
	Hist       [10]ReaderHistItem
	Start, Len int
	mtx        sync.RWMutex
}

type ReaderHistItem struct {
	Sha256 string
	Name   string
	Data   []byte
}

func loadHistFromFile() {
	if debug {
		fmt.Println("INFO: NO loading hist in debeg mode!")
		return
	}
	data, err := os.ReadFile(readerHistFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("WARN: could not read:", readerHistFile, "reason:", err)
		}
		return
	}

	if err = json.Unmarshal(data, &readerHist); err != nil {
		fmt.Println("WARN: could not parse json:", err)
		return
	}
	fmt.Println("INFO: loaded history form:", readerHistFile)
}

func (rh *ReaderHist) saveToFile() {
	if debug {
		return
	}
	data, err := json.Marshal(rh)
	if err != nil {
		fmt.Println("WARN: could not make json:", err)
		return
	}

	f, err := os.Create(readerHistFile)
	if err != nil {
		fmt.Println("WARN: could not create:", readerHistFile, "reason:", err)
		return
	}
	defer f.Close()
	if _, err = f.Write(data); err != nil {
		fmt.Println("WARN: could not write to:", readerHistFile, "reason:", err)
		return
	}
}

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

			if dict.ContainsArabic(l) {
				f = false
				pageName = l
			}
		}
		for _, w := range strings.Split(l, " ") {
			if w != "" {
				wr := s.d.FindWord(w)
				n := dict.ContainsArabic(w)
				cp = append(cp, ReaderWord{n, w, wr})
			}
		}
		reader = append(reader, cp)
	}

	readerData := ReaderData{pageName, reader}
	buf := new(bytes.Buffer)
	if err := t.ExecuteTemplate(buf, "reader.html", &readerData); debug && err != nil {
		panic(err)
	}
	w.Write(buf.Bytes())

	if r.FormValue("save") != "on" {
		return
	}

	// save stuff in a seperate routine
	go func() {
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
	}()
}
