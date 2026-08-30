package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"unicode"
)

var pngSig = []byte{137, 80, 78, 71, 13, 10, 26, 10}
var exifFields = map[uint16]string{0x010e: "comments", 0x010f: "make", 0x0110: "model", 0x013b: "author", 0x8298: "author"}
var xmpFields = map[xml.Name]string{{Space: "http://ns.adobe.com/xap/1.0/", Local: "CreateDate"}: "capture", {Space: "http://ns.adobe.com/tiff/1.0/", Local: "Make"}: "make", {Space: "http://ns.adobe.com/tiff/1.0/", Local: "Model"}: "model", {Space: "http://ns.adobe.com/exif/1.0/", Local: "DateTimeOriginal"}: "capture", {Space: "http://purl.org/dc/elements/1.1/", Local: "creator"}: "author", {Space: "http://purl.org/dc/elements/1.1/", Local: "rights"}: "author", {Space: "http://purl.org/dc/elements/1.1/", Local: "description"}: "comments"}
var pngFields = map[string]string{"Author": "author", "Artist": "author", "Copyright": "author", "Description": "comments", "Comment": "comments", "Caption": "comments", "Creation Time": "capture"}

type metadata map[string][]string
type report struct {
	name, format, hash string
	size               int64
	width, height      int
	m                  metadata
}
type entry struct {
	tag, typ uint16
	data     []byte
}

func (m metadata) categories() (c []string) {
	for _, x := range []struct {
		name string
		ok   bool
	}{{"location", len(m["location"]) > 0}, {"capture time", len(m["capture"]) > 0}, {"device", len(m["make"])+len(m["model"]) > 0}, {"authorship", len(m["author"]) > 0}, {"comments", len(m["comments"]) > 0}} {
		if x.ok {
			c = append(c, x.name)
		}
	}
	return
}

func inspect(path string) (report, error) {
	var r report
	st, err := os.Lstat(path)
	if err != nil {
		return r, err
	}
	if !st.Mode().IsRegular() {
		return r, fmt.Errorf("expected one regular JPEG or PNG file")
	}
	if st.Size() > 100<<20 {
		return r, fmt.Errorf("file exceeds 100 MiB limit")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || format != "jpeg" && format != "png" {
		return r, fmt.Errorf("unsupported or malformed image; expected JPEG or PNG")
	}
	sum := sha256.Sum256(b)
	r = report{name: clean(path[strings.LastIndex(path, "/")+1:]), format: strings.ToUpper(format), hash: fmt.Sprintf("%x", sum), size: int64(len(b)), width: cfg.Width, height: cfg.Height, m: metadata{}}
	if format == "jpeg" {
		scanJPEG(b, r.m)
	} else {
		scanPNG(b, r.m)
	}
	return r, nil
}

func scanJPEG(b []byte, m metadata) {
	for p := 2; p+4 <= len(b) && b[p] == 0xff; {
		for p < len(b) && b[p] == 0xff {
			p++
		}
		if p >= len(b) {
			return
		}
		mark := b[p]
		p++
		if mark == 0xda || mark == 0xd9 {
			return
		}
		if mark == 1 || mark >= 0xd0 && mark <= 0xd7 {
			continue
		}
		if p+2 > len(b) {
			return
		}
		n := int(binary.BigEndian.Uint16(b[p : p+2]))
		if n < 2 || p+n > len(b) {
			return
		}
		v := b[p+2 : p+n]
		p += n
		if mark == 0xe1 && bytes.HasPrefix(v, []byte("Exif\x00\x00")) {
			add(m, "containers", "JPEG EXIF")
			parseEXIF(v[6:], m)
		} else if mark == 0xe1 && bytes.HasPrefix(v, []byte("http://ns.adobe.com/xap/1.0/\x00")) {
			add(m, "containers", "JPEG XMP")
			parseXMP(v[29:], m)
		} else if mark == 0xfe {
			add(m, "containers", "JPEG COM")
			add(m, "comments", string(v))
		}
	}
}

func scanPNG(b []byte, m metadata) {
	if !bytes.HasPrefix(b, pngSig) {
		return
	}
	for p := 8; p+12 <= len(b); {
		n := int(binary.BigEndian.Uint32(b[p : p+4]))
		end := p + 12 + n
		if n < 0 || end > len(b) {
			return
		}
		typ := string(b[p+4 : p+8])
		v := b[p+8 : p+8+n]
		switch typ {
		case "eXIf":
			add(m, "containers", "PNG eXIf")
			parseEXIF(v, m)
		case "tEXt":
			add(m, "containers", "PNG tEXt")
			pngText(v, m)
		case "iTXt":
			add(m, "containers", "PNG iTXt")
			iTXt(v, m)
		}
		p = end
		if typ == "IEND" {
			return
		}
	}
}

func pngText(b []byte, m metadata) {
	i := bytes.IndexByte(b, 0)
	if i > 0 {
		storeText(string(b[:i]), latin1(b[i+1:]), m)
	}
}
func iTXt(b []byte, m metadata) {
	i := bytes.IndexByte(b, 0)
	if i <= 0 || i+3 > len(b) || b[i+1] != 0 || b[i+2] != 0 {
		return
	}
	key := string(b[:i])
	r := b[i+3:]
	for range 2 {
		j := bytes.IndexByte(r, 0)
		if j < 0 {
			return
		}
		r = r[j+1:]
	}
	storeText(key, string(r), m)
}
func storeText(k, v string, m metadata) {
	k = clean(k)
	if k == "XML:com.adobe.xmp" {
		add(m, "containers", "PNG XMP")
		parseXMP([]byte(v), m)
		return
	}
	if field := pngFields[k]; field != "" {
		add(m, field, v)
	}
}

func parseEXIF(b []byte, m metadata) {
	if len(b) < 8 {
		return
	}
	var o binary.ByteOrder
	if string(b[:2]) == "II" {
		o = binary.LittleEndian
	} else if string(b[:2]) == "MM" {
		o = binary.BigEndian
	} else {
		return
	}
	if o.Uint16(b[2:4]) != 42 {
		return
	}
	var exif, gps uint32
	for _, e := range ifd(b, o, o.Uint32(b[4:8])) {
		if field := exifFields[e.tag]; field != "" {
			add(m, field, text(e.data))
		}
		if e.tag == 0x8769 {
			exif = uintValue(e, o)
		}
		if e.tag == 0x8825 {
			gps = uintValue(e, o)
		}
	}
	for _, e := range ifd(b, o, exif) {
		if e.tag == 0x9003 || e.tag == 0x9004 {
			add(m, "capture", text(e.data))
		}
	}
	var lr, or string
	var lat, lon []float64
	for _, e := range ifd(b, o, gps) {
		switch e.tag {
		case 1:
			if e.typ == 2 { lr = text(e.data) }
		case 2:
			if e.typ == 5 { lat = rationals(e.data, o) }
		case 3:
			if e.typ == 2 { or = text(e.data) }
		case 4:
			if e.typ == 5 { lon = rationals(e.data, o) }
		}
	}
	if (lr == "N" || lr == "S") && (or == "E" || or == "W") {
		a, aok := dms(lat, 90); z, zok := dms(lon, 180)
		if lr == "S" { a = -a }; if or == "W" { z = -z }
		if aok && zok { add(m, "location", fmt.Sprintf("%.6f, %.6f", a, z)) }
	}
}
func ifd(b []byte, o binary.ByteOrder, off uint32) (out []entry) {
	p := int(off)
	if off == 0 || p+2 > len(b) {
		return
	}
	n := int(o.Uint16(b[p : p+2]))
	if n > 256 {
		return
	}
	sizes := map[uint16]int{1: 1, 2: 1, 3: 2, 4: 4, 5: 8, 7: 1, 9: 4, 10: 8}
	for i := 0; i < n; i++ {
		q := p + 2 + i*12
		if q+12 > len(b) {
			return
		}
		typ := o.Uint16(b[q+2 : q+4])
		sz := sizes[typ] * int(o.Uint32(b[q+4:q+8]))
		if sz <= 0 || sz > 1<<20 {
			continue
		}
		var v []byte
		if sz <= 4 {
			v = b[q+8 : q+8+sz]
		} else {
			x := int(o.Uint32(b[q+8 : q+12]))
			if x < 0 || x+sz > len(b) {
				continue
			}
			v = b[x : x+sz]
		}
		out = append(out, entry{o.Uint16(b[q : q+2]), typ, v})
	}
	return
}
func uintValue(e entry, o binary.ByteOrder) uint32 {
	if e.typ == 3 && len(e.data) >= 2 {
		return uint32(o.Uint16(e.data))
	}
	if len(e.data) >= 4 {
		return o.Uint32(e.data)
	}
	return 0
}
func rationals(v []byte, o binary.ByteOrder) (r []float64) {
	for len(v) >= 8 {
		a, d := o.Uint32(v[:4]), o.Uint32(v[4:8])
		if d == 0 {
			return nil
		}
		r = append(r, float64(a)/float64(d))
		v = v[8:]
	}
	return
}
func dms(v []float64, limit float64) (float64, bool) {
	if len(v) != 3 || v[0] < 0 || v[1] < 0 || v[1] >= 60 || v[2] < 0 || v[2] >= 60 { return 0, false }
	n := v[0] + v[1]/60 + v[2]/3600
	return n, n <= limit
}
func text(v []byte) string { return string(bytes.Trim(v, "\x00")) }
func latin1(v []byte) string {
	r := make([]rune, len(v)); for i, b := range v { r[i] = rune(b) }; return string(r)
}

func parseXMP(b []byte, m metadata) {
	d := xml.NewDecoder(bytes.NewReader(b))
	current := ""
	for {
		tok, err := d.Token()
		if err != nil {
			return
		}
		switch x := tok.(type) {
		case xml.StartElement:
			if k := xmpFields[x.Name]; k != "" {
				current = k
			}
			for _, a := range x.Attr {
				if k := xmpFields[a.Name]; k != "" {
					add(m, k, a.Value)
				}
			}
		case xml.CharData:
			if current != "" {
				add(m, current, string(x))
			}
		case xml.EndElement:
			if xmpFields[x.Name] != "" {
				current = ""
			}
		}
	}
}

func add(m metadata, key, s string) {
	s = clean(s)
	if s == "" {
		return
	}
	for _, v := range m[key] {
		if strings.EqualFold(v, s) {
			return
		}
	}
	if len(m[key]) < 32 {
		m[key] = append(m[key], s)
	}
}
func clean(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			b.WriteByte(' ')
		} else {
			b.WriteRune(r)
		}
	}
	s = strings.Join(strings.Fields(b.String()), " ")
	r := []rune(s)
	if len(r) > 300 {
		s = string(r[:300]) + "…"
	}
	return s
}
