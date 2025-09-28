package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/wizsk/dict/dict"
)

var (
	readerHistDir = func() string {
		n := ""
		if h, err := os.UserHomeDir(); err == nil {
			n = filepath.Join(h, ".dict_history")
		} else {
			n = "dict_history"
		}
		if _, err := os.Stat(n); err != nil {
			if err = os.Mkdir(n, 0700); err != nil && !os.IsExist(err) {
				return ""
			}
		}
		fmt.Printf("INFO: Permanent hist dir: %q\n", n)
		return n
	}()
	readerTmpDir = func() string {
		n := filepath.Join(os.TempDir(), "dict_history")
		if _, err := os.Stat(n); err != nil {
			if err = os.Mkdir(n, 0700); err != nil && !os.IsExist(err) {
				return ""
			}
		}
		fmt.Printf("INFO: Temporary hist dir: %q\n", n)
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

func (s *server) readerHandler(w http.ResponseWriter, r *http.Request) {
	t := s.t
	if debug {
		t = p(template.ParseGlob("pub/*.html"))
	}

	txt := strings.TrimSpace(r.FormValue("txt"))
	if txt == "" {
		h := strings.TrimPrefix(r.URL.EscapedPath(), "/rd/")
		if h == "" {
			var s strings.Builder
			var dir []os.DirEntry
			if readerHistDir != "" {
				dir, _ = os.ReadDir(readerHistDir)
			}
			if len(dir) > 0 {
				s.WriteString(
					"<div>الملفات الدائمة</div>",
				)
			}
			for _, d := range dir {
				name := strings.SplitN(d.Name(), "__", 2)[1]
				name, err := url.PathUnescape(name)
				if err != nil {
					name = "؟؟؟؟؟"
				}
				a := fmt.Sprintf(
					`<a class="hist-item" href="/rd/%s?perm=true">- %s</a>`,
					d.Name(),
					html.EscapeString(name))
				s.WriteString(a)
			}
			dir, _ = os.ReadDir(readerTmpDir)
			if len(dir) > 0 {
				s.WriteString(
					"<div>الملفات المؤقتة</div>",
				)
			}
			for _, d := range dir {
				name := strings.SplitN(d.Name(), "__", 2)[1]
				name, err := url.PathUnescape(name)
				if err != nil {
					name = "؟؟؟؟؟"
				}
				a := fmt.Sprintf(
					`<a class="hist-item" href="/rd/%s">- %s</a>`,
					d.Name(),
					html.EscapeString(name))
				s.WriteString(a)
			}
			if err := t.ExecuteTemplate(w, "readerInpt.html",
				template.HTML(s.String())); debug && err != nil {
				panic(err)
			}
			return
		}
		d := readerTmpDir
		if r.FormValue("perm") == "true" {
			d = readerHistDir
		}
		if d == "" {
			http.Redirect(w, r, "/rd/", http.StatusMovedPermanently)
			return
		}
		dirs, _ := os.ReadDir(d)
		for _, dir := range dirs {
			fmt.Println(dir.Name() == h, dir.Name(), h)
			if dir.Name() == h {
				f, err := os.Open(filepath.Join(d, dir.Name()))
				if err != nil {
					http.Redirect(w, r, "/rd/", http.StatusMovedPermanently)
					return
				}
				defer f.Close()
				io.Copy(w, f)
				return
			}
		}

		http.Redirect(w, r, "/rd/", http.StatusMovedPermanently)
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
	data := new(bytes.Buffer)
	if err := t.ExecuteTemplate(data, "reader.html", &readerData); debug && err != nil {
		panic(err)
	}
	buf := bytes.NewReader(data.Bytes())
	io.Copy(w, buf)

	isSave := r.FormValue("save") == "on"
	go func() {
		d := readerTmpDir
		if isSave && readerHistDir != "" {
			d = readerHistDir
		}
		sha := fmt.Sprintf("%x", sha256.Sum256([]byte(txt)))
		name := url.PathEscape(pageName)
		f := filepath.Join(d, sha+"__"+name)
		file, err := os.Create(f)
		if err != nil {
			fmt.Printf("WARN: err: %v\n", err)
			return
		}
		defer file.Close()
		buf.Seek(0, io.SeekStart)
		i, _ := io.Copy(file, buf)
		fmt.Printf("INFO: saved %d: %q\n", i, f)
	}()
}
