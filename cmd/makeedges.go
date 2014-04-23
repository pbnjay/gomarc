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

var outfilename = flag.String("o", "agricola2nalt.data", "output file name")

// control number
var primaryId = "016a"

// primary, general, geographic topics + geograpic subject
var secondaryIds = []string{"650a", "650x", "650z", "651a"}

// most of the code in here is because NAL's QC sucks...
// sometimes annotations end with a '.'...
// sometimes annotations are duplicated...
func edgesFromRecord(r *gomarc.Reader) (ret []string) {
	agricolaId, valid := r.GetField(primaryId[:3], primaryId[3:])
	if !valid {
		return nil
	}

	nodupes := make(map[string]bool)
	for _, sub := range secondaryIds {
		flds, exists := r.GetFields(sub[:3], sub[3:])
		if !exists {
			continue
		}
		for _, fld := range flds {
			if strings.HasSuffix(fld, ".") {
				if strings.HasSuffix(fld, "etc.") {
					// skip
				} else if fld[len(fld)-2] == ')' || fld[len(fld)-3] != '.' { // acronym? "U.S."
					fld = fld[:len(fld)-1]
				}
			}
			if !nodupes[fld] {
				ret = append(ret, agricolaId+"\t"+fld)
				nodupes[fld] = true
			}
		}
	}
	return ret
}

func processFile(filename string, fout *os.File) error {
	fmt.Printf("Processing '%s'...\n", filename)

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	nrec := 0
	mr := gomarc.NewReader(f)
	for mr.Next() {
		edges := edgesFromRecord(mr)
		if edges != nil {
			fmt.Fprintln(fout, strings.Join(edges, "\n"))
			nrec += len(edges)
		}
	}
	if mr.Err != nil {
		return mr.Err
	}
	fmt.Printf("  %d edges\n", nrec)
	return nil
}

func main() {
	flag.Parse()
	fout, err := os.OpenFile(*outfilename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer fout.Close()

	for _, fn := range flag.Args() {
		err = processFile(fn, fout)
		if err != nil {
			fmt.Println(err)
		}
	}
}
