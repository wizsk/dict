package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type dictionary struct {
	dictPref  dictEntries
	dictStems dictEntries
	dictSuff  dictEntries

	tableAB tableEntries
	tableBC tableEntries
	tableAC tableEntries
}

type Entry struct {
	Root  string
	Word  string
	Morph string
	Def   string
	Fam   string
	Pos   string
}

type dictEntries map[string][]Entry
type tableEntries map[string][]string

func (d *dictionary) findWord(word string) []Entry {
	res := []Entry{}
	w := []rune(transliterateRmHarakats(word))

	for i := 0; i < len(w); i++ {
		for j := i + 1; j <= len(w); j++ {
			// fmt.Println(rSlice(w, 0, i), rSlice(w, i, j), rSlice(w, j, len(w)))
			c := d.dict(rSlice(w, 0, i), rSlice(w, i, j), rSlice(w, j, len(w)))
			res = append(res, c...)
		}
	}
	return res
}

func rSlice(r []rune, start, end int) string {
	return string(r[start:end])

}
func bracketify(word string, space int) string {
	if word != "" && word[0] != '[' {
		if space == 1 {
			return " [" + word + "]"
		} else if space == 2 {
			return "[" + word + "] "
		} else {
			return "[" + word + "]"
		}
	} else {
		return ""
	}
}

func formatSuffix(s string) string {
	p := strings.Split(s, "<pos>")
	return strings.TrimSpace(p[0])
}

func (d *dictionary) dict(pref, stem, suff string) []Entry {
	prf := d.dictPref[pref]
	stm := d.dictStems[stem]
	suf := d.dictSuff[suff]

	res := []Entry{}

	for _, p := range prf {
		for _, s := range stm {
			for _, su := range suf {
				if !d.obeysGrammer(p.Morph, s.Morph, su.Morph) {
					continue
				}

				c := Entry{
					Root: deTransliterate(s.Root),
					Word: deTransliterate(p.Word + s.Word + su.Word),
					Def:  bracketify(p.Def, 2) + s.Def + bracketify(su.Def, 1),
					Fam:  s.Fam,
				}
				fmt.Println("?", formatSuffix(p.Def), formatSuffix(su.Def), s.Def)
				res = append(res, c)
				// fmt.Printf("root: %s, word: %s, def: %s, pos: %s, fam: %s, morpth: %s\n",
				// 	deTransliterate(s.Root),
				// 	deTransliterate(p.Word+s.Word+su.Word),
				// 	strings.Join([]string{p.Def, s.Def, su.Def}, "|"),
				// 	strings.Join([]string{p.Pos, s.Pos, su.Pos}, ", "),
				// 	s.Fam,
				// 	strings.Join([]string{p.Morph, s.Morph, su.Morph}, ", "),
				// )
			}
		}
	}
	return res
}

func (d *dictionary) obeysGrammer(pref, stem, suff string) bool {
	return slices.Index(d.tableAB[pref], stem) != -1 &&
		slices.Index(d.tableBC[stem], suff) != -1 &&
		slices.Index(d.tableAC[pref], suff) != -1

}

func parseTable(f string) tableEntries {
	data := p(os.ReadFile(f))
	en := map[string][]string{}
	lines := bufio.NewScanner(bytes.NewBuffer(data))
	lineC := 0
	for lines.Scan() {
		lineC++
		// l := strings.TrimSpace(lines.Text())
		l := lines.Text()
		if len(l) == 0 || l[0] == ';' {
			continue
		}
		parts := strings.Split(l, " ")
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "parseDict: %s:%d: bad entry of %d: %s\n",
				f, lineC, len(parts), l)
			continue
		}
		m := parts[1]
		en[parts[0]] = append(en[parts[0]], m)
	}
	return en
}

func p[T any](r T, err error) T {
	if err != nil {
		panic(err)
	}
	return r
}

func parseDict(f string) dictEntries {
	data := p(os.ReadFile(f))
	en := map[string][]Entry{}

	root := ""
	family := ""
	lines := bufio.NewScanner(bytes.NewBuffer(data))
	lineC := 0
	for lines.Scan() {
		lineC++
		// l := strings.TrimSpace(lines.Text())
		l := lines.Text()
		// blank line && comments
		if len(l) == 0 || strings.TrimSpace(l) == ";" {
			continue
		}

		if strings.Contains(l, ";--- ") {
			root = strings.Split(l, " ")[1]
		} else if strings.Contains(l, "; form") {
			family = strings.Split(l, " ")[2]
		} else if l[0] != ';' {
			parts := strings.SplitN(l, "\t", 4)

			// meta := strings.Split(strings.Join(parts[3:], " "), "<pos>")
			// def := strings.ReplaceAll(meta[0], ";", ", ")
			// pos := ""
			// if len(meta) >= 2 {
			// 	pos = strings.Split(meta[1], "</pos>")[0]
			// }
			// // e := Entry{Root: deTransliterate(root), Word: deTransliterate(parts[1]), Morph: parts[2], Def: def, Fam: family, Pos: pos}
			// e := Entry{
			// 	Root: root, Word: parts[1],
			// 	Morph: parts[2], Def: def,
			// 	Fam: family, Pos: pos,
			// }
			e := Entry{
				Root: root, Word: parts[1],
				Morph: parts[2], Def: parts[3],
				Fam: family,
			}

			en[parts[0]] = append(en[parts[0]], e)
		}
	}

	return en
}

func main() {
	const dataRoot = "data"
	dicts := []string{"dictprefixes", "dictstems", "dictsuffixes"}
	tables := []string{"tableab", "tableac", "tablebc"}

	dict := dictionary{}

	dict.dictPref = parseDict(filepath.Join(dataRoot, dicts[0]))
	dict.dictStems = parseDict(filepath.Join(dataRoot, dicts[1]))
	dict.dictSuff = parseDict(filepath.Join(dataRoot, dicts[2]))

	dict.tableAB = parseTable(filepath.Join(dataRoot, tables[0]))
	dict.tableAC = parseTable(filepath.Join(dataRoot, tables[1]))
	dict.tableBC = parseTable(filepath.Join(dataRoot, tables[2]))

	for _, e := range dict.findWord(os.Args[1]) {
		fmt.Println(e)
	}
}
