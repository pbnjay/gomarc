// Package gomarc implements a reader for files in the MARC 21 bibliographic format.
//
// This package borrows many concepts from the excellent pymarc package:
//     https://github.com/edsu/pymarc
//
package gomarc

import (
	"fmt"
	"io"
	"strings"
)

// Reader is a MARC21-format reader which can enumerate records in an io.reader
type Reader struct {
	f       io.Reader
	current *marcRecord
	Err     error
}

type marcField struct {
	// length-2 string [0]=first indicator, [1]=second indicator
	indicators string
	// contents[][0] = subfield code
	// contents[][1:] = contents
	contents []string
}

type marcRecord struct {
	// leader string (24 chars)
	leader string

	// key = tag = 3-char string
	fields map[string][]marcField
}

// NewReader creates a new gomarc.Reader with the given io.Reader
func NewReader(rio io.Reader) *Reader {
	return &Reader{f: rio}
}

// Next advances to the next record and returns true if one exists, or false if
// EOF is encountered (outside of a record).
func (r *Reader) Next() bool {
	// read the record length
	var first5 = []byte{0, 0, 0, 0, 0}
	n, err := io.ReadFull(r.f, first5)
	if err != nil {
		if err == io.EOF {
			return false
		}
		r.Err = fmt.Errorf("invalid reading record length (err=%v)", err.Error())
		return false
	}

	// parse record length as 5-digit 0-padded character numeric
	rlen := 0
	n, err = fmt.Sscanf(string(first5), "%05d", &rlen)
	if err != nil || n != 1 {
		r.Err = fmt.Errorf("invalid record length: '%s'", string(first5))
		return false
	}

	recordBytes := make([]byte, rlen, rlen)
	copy(recordBytes, first5)
	n, err = io.ReadFull(r.f, recordBytes[5:])
	if err != nil || n != (rlen-5) {
		r.Err = fmt.Errorf("invalid reading record contents (err=%v)", err.Error())
		return false
	}

	r.current, r.Err = parseRecord(recordBytes)
	return (r.Err == nil)
}

func parseRecord(data []byte) (*marcRecord, error) {
	rec := &marcRecord{}
	rec.leader = string(data[0:24])
	rec.fields = make(map[string][]marcField)

	// extract base address
	var base int
	_, err := fmt.Sscanf(string(data[12:17]), "%05d", &base)
	if err != nil {
		return nil, err
	}
	if base <= 0 || base >= len(data) {
		return nil, fmt.Errorf("invalid base address in marc record")
	}

	directory := data[24 : base-1]
	if (len(directory) % 12) != 0 {
		return nil, fmt.Errorf("invalid directory format in marc record")
	}
	for o := 0; o < len(directory); o += 12 {
		etag := string(directory[o : o+3])
		elen := string(directory[o+3 : o+7])
		entryLength := 0
		eoff := string(directory[o+7 : o+12])
		entryOffset := 0
		fmt.Sscanf(elen, "%04d", &entryLength)
		fmt.Sscanf(eoff, "%05d", &entryOffset)
		edata := string(data[base+entryOffset : base+entryOffset+entryLength-1])

		fld := marcField{}
		if etag < "010" {
			// control field
			fld.indicators = ""
			fld.contents = []string{edata}

		} else {
			subs := strings.Split(edata, "\x1F")
			switch len(subs[0]) {
			case 0:
				fld.indicators = "  "
			case 1:
				fld.indicators = subs[0] + " "
			case 2:
				fld.indicators = subs[0]
			default:
				fld.indicators = subs[0][:2]
			}

			for si, sf := range subs {
				if len(sf) == 0 || si == 0 {
					continue
				}
				fld.contents = append(fld.contents, sf)
			}
		}
		rec.fields[etag] = append(rec.fields[etag], fld)
	}
	return rec, nil
}

// GetField returns the first field in tag with the given subfield indicator. If
// subfield=="", then it returns the contents of the first subfield.
func (r *Reader) GetField(tag, subfield string) (string, bool) {
	if r.current == nil {
		return "", false
	}
	fld, hasField := r.current.fields[tag]
	if !hasField {
		return "", false
	}

	if tag < "010" {
		// control field, ignore subfield
		return fld[0].contents[0], true
	}

	if subfield == "" {
		return fld[0].contents[0][1:], true
	}
	for _, f := range fld {
		for _, sf := range f.contents {
			if strings.HasPrefix(sf, subfield) {
				return sf[1:], true
			}
		}
	}

	return "", false
}

// GetFields returns a slice of fields in the tag with the given subfield
// indicator. If subfield=="", then it returns the contents of all subfields.
func (r *Reader) GetFields(tag, subfield string) ([]string, bool) {
	fld, hasField := r.current.fields[tag]
	if !hasField {
		return nil, false
	}

	if tag < "010" {
		// control field, ignore subfield
		return fld[0].contents, true
	}

	ret := []string{}
	if subfield == "" {
		for _, f := range fld {
			for _, sf := range f.contents {
				ret = append(ret, sf[1:])
			}
		}
	} else {
		for _, f := range fld {
			for _, sf := range f.contents {
				if strings.HasPrefix(sf, subfield) {
					ret = append(ret, sf[1:])
				}
			}
		}
	}

	return ret, len(ret) > 0
}
