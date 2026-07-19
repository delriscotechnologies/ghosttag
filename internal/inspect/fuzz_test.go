package inspect

import (
	"encoding/binary"
	"testing"

	"github.com/delriscotechnologies/ghosttag/internal/model"
)

func FuzzImageParsersDoNotPanic(f *testing.F) {
	jpeg := []byte{0xff, 0xd8}
	jpeg = append(jpeg, jpegSegment(0xc0, []byte{8, 0, 1, 0, 1, 1, 1, 0x11, 0})...)
	jpeg = append(jpeg, 0xff, 0xd9)

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], 1)
	binary.BigEndian.PutUint32(ihdr[4:8], 1)
	png := append([]byte{}, pngSignature...)
	png = append(png, testChunk("IHDR", ihdr)...)
	png = append(png, testChunk("IEND", nil)...)

	f.Add(jpeg)
	f.Add(png)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1024*1024 {
			t.Skip()
		}
		format, err := detectFormat(data)
		if err != nil {
			return
		}
		var metadata model.Metadata
		var warnings []string
		collector := newCollector(&metadata, &warnings)
		switch format {
		case "JPEG":
			_, _, _ = parseJPEG(data, collector)
		case "PNG":
			_, _, _ = parsePNG(data, collector)
		}
	})
}

func FuzzMetadataParsersDoNotPanic(f *testing.F) {
	validXMP := []byte(`<x:xmpmeta xmlns:x="adobe:ns:meta/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:xmp="http://ns.adobe.com/xap/1.0/" xmlns:exif="http://ns.adobe.com/exif/1.0/"><rdf:RDF><rdf:Description xmp:CreateDate="2026-07-18T12:34:56Z" exif:GPSLatitude="37N" exif:GPSLongitude="122W"/></rdf:RDF></x:xmpmeta>`)
	f.Add(byte(0), testTIFF())
	f.Add(byte(1), validXMP)
	f.Add(byte(2), []byte("Author\x00Example"))
	f.Fuzz(func(t *testing.T, parser byte, data []byte) {
		if len(data) > 1024*1024 {
			t.Skip()
		}
		var metadata model.Metadata
		var warnings []string
		collector := newCollector(&metadata, &warnings)
		switch parser % 3 {
		case 0:
			_ = parseEXIF(data, "fuzz EXIF", collector)
		case 1:
			_ = parseXMP(data, "fuzz XMP", collector)
		case 2:
			keyword, value, err := parseTextChunk(data)
			if err == nil {
				parsePNGText(keyword, value, "tEXt", collector)
			}
		}
	})
}

func FuzzXMPCoordinatesDoNotPanic(f *testing.F) {
	f.Add("37,46.5N")
	f.Add("NaN")
	f.Fuzz(func(t *testing.T, value string) {
		_, _ = parseXMPCoordinate(value, true)
	})
}
