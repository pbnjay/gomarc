// A simple example XML producer from MARC records, which works great with
// records from the AGRICOLA database: http://agricola.nal.usda.gov/

package main

import (
	"flag"
	"fmt"
	"github.com/pbnjay/gomarc"
	"os"
	"strings"
)

var outfilename = flag.String("o", "output.xml", "output file name")

var xmlFieldMap = map[string]string{
	"control-number":         "016a",
	"language":               "008",
	"title/main":             "245a",
	"title/subtitle":         "245b",
	"abstract":               "520a",
	"authors/primary-author": "100a",
	"authors/author":         "700a",
	"journal/title":          "773t",
	"journal/publisher":      "773d",
	"journal/volume-issue":   "773g",
	"extent":                 "300a",
	"isbn":                   "020a",
	"genre-form":             "655a",
	"url":                    "856u",
	"doi":                    "0242",
	"subjects/geographic":       "651a",
	"subjects/primary-topic":    "650a",
	"subjects/general-topic":    "650x",
	"subjects/geographic-topic": "650z",
	"published":                 "260",
}

func xmlRecord(r *gomarc.Reader) string {
	xmldata := make(map[string][]string)
	for name, tagsf := range xmlFieldMap {
		tag := tagsf[:3]
		sf := tagsf[3:]
		flds, hadFlds := r.GetFields(tag, sf)
		if hadFlds {
			switch name {
			case "language":
				flds = []string{flds[0][35:38]}
			case "published":
				flds = []string{strings.Join(flds, " ")}
			}

			xmldata[name] = append(xmldata[name], flds...)
		}
	}
	xmlout := make(map[string]string)
	for tagname, contents := range xmldata {
		maintag := ""
		indent := "  "
		if strings.Contains(tagname, "/") {
			parts := strings.Split(tagname, "/")
			maintag = parts[0]
			tagname = parts[1]
			indent = "    "
		}
		nodupes := make(map[string]bool)
		for _, c := range contents {
			if !nodupes[c] {
				xmlout[maintag] += fmt.Sprintf("%s<%s>%s</%s>\n", indent, tagname, c, tagname)
				nodupes[c] = true
			}
		}
	}
	xmlstring := xmlout[""]
	for othertag, inner := range xmlout {
		if othertag == "" {
			continue
		}
		xmlstring += fmt.Sprintf("  <%s>\n%s  </%s>\n", othertag, inner, othertag)
	}
	return "<document>\n" + xmlstring + "</document>"
}

func main() {
	flag.Parse()
	fmt.Println(flag.Arg(0))
	fout, err := os.OpenFile(*outfilename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer fout.Close()
	f, err := os.Open(flag.Arg(0))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprintln(fout, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintln(fout, "<documentset>")
	mr := gomarc.NewReader(f)
	for mr.Next() {
		xml := xmlRecord(mr)
		fmt.Fprintln(fout, xml)
	}
	fmt.Fprintln(fout, "</documentset>")
	if mr.Err != nil {
		panic(mr.Err)
	}
}
