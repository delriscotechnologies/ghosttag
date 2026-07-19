package inspect

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectPNGTextAndXMP(t *testing.T) {
	base := encodePNG(t, 2, 3)
	xmp := []byte(`<x:xmpmeta xmlns:x="adobe:ns:meta/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:xmp="http://ns.adobe.com/xap/1.0/" xmlns:tiff="http://ns.adobe.com/tiff/1.0/" xmlns:exif="http://ns.adobe.com/exif/1.0/"><rdf:RDF><rdf:Description xmp:CreateDate="2026-07-18T12:34:56Z" xmp:CreatorTool="ghost editor" tiff:Make="Acme Corp" exif:GPSLatitude="37,46.5N" exif:GPSLongitude="122,25W"/></rdf:RDF></x:xmpmeta>`)

	chunks := [][]byte{
		pngChunk("tEXt", append([]byte("Author\x00"), []byte("A. Person")...)),
		pngChunk("tEXt", append([]byte("Description\x00"), []byte("Shared at a meetup")...)),
		pngChunk("zTXt", compressedPNGText(t, "Software", "PNG Writer")),
		pngChunk("iTXt", internationalPNGText("XML:com.adobe.xmp", xmp)),
	}
	data := insertBeforeIEND(base, chunks...)
	path := filepath.Join(t.TempDir(), "sample.png")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := File(path)
	if err != nil {
		t.Fatalf("File returned an error: %v", err)
	}
	if result.File.DetectedFormat != "PNG" || result.File.Width != 2 || result.File.Height != 3 {
		t.Fatalf("unexpected file facts: %+v", result.File)
	}
	if len(result.Metadata.Authors) != 1 || result.Metadata.Authors[0].Value != "A. Person" {
		t.Fatalf("unexpected authors: %+v", result.Metadata.Authors)
	}
	if len(result.Metadata.Comments) != 1 || result.Metadata.Comments[0].Value != "Shared at a meetup" {
		t.Fatalf("unexpected comments: %+v", result.Metadata.Comments)
	}
	if len(result.Metadata.CaptureTime) != 1 || result.Metadata.CaptureTime[0].Value != "2026-07-18T12:34:56Z" {
		t.Fatalf("unexpected capture time: %+v", result.Metadata.CaptureTime)
	}
	if len(result.Metadata.DeviceMake) != 1 || result.Metadata.DeviceMake[0].Value != "Acme Corp" {
		t.Fatalf("unexpected device make: %+v", result.Metadata.DeviceMake)
	}
	if len(result.Metadata.Locations) != 1 || math.Abs(result.Metadata.Locations[0].Latitude-37.775) > 0.000001 {
		t.Fatalf("unexpected XMP location: %+v", result.Metadata.Locations)
	}
	if len(result.Metadata.Software) != 2 {
		t.Fatalf("unexpected software values: %+v", result.Metadata.Software)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", result.Warnings)
	}
}

func TestInspectJPEGEXIFAndExtensionMismatch(t *testing.T) {
	tiff := testTIFF()
	app1Payload := append(append([]byte{}, exifHeader...), tiff...)
	data := []byte{0xff, 0xd8}
	data = append(data, jpegSegment(jpegAPP1, app1Payload)...)
	data = append(data, jpegSegment(jpegCOM, []byte("JPEG comment"))...)
	data = append(data, jpegSegment(0xc0, []byte{8, 0, 2, 0, 3, 1, 1, 0x11, 0})...)
	data = append(data, 0xff, 0xd9)

	path := filepath.Join(t.TempDir(), "renamed.png")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := File(path)
	if err != nil {
		t.Fatalf("File returned an error: %v", err)
	}
	if result.File.DetectedFormat != "JPEG" || result.File.Width != 3 || result.File.Height != 2 {
		t.Fatalf("unexpected file facts: %+v", result.File)
	}
	if len(result.Metadata.DeviceMake) != 1 || result.Metadata.DeviceMake[0].Value != "Acme Corp" {
		t.Fatalf("unexpected device make: %+v", result.Metadata.DeviceMake)
	}
	if len(result.Metadata.DeviceModel) != 1 || result.Metadata.DeviceModel[0].Value != "Model One" {
		t.Fatalf("unexpected device model: %+v", result.Metadata.DeviceModel)
	}
	if len(result.Metadata.CaptureTime) != 1 || !strings.HasPrefix(result.Metadata.CaptureTime[0].Value, "2026-07-18T12:34:56") {
		t.Fatalf("unexpected capture time: %+v", result.Metadata.CaptureTime)
	}
	if len(result.Metadata.Locations) != 1 {
		t.Fatalf("unexpected locations: %+v", result.Metadata.Locations)
	}
	location := result.Metadata.Locations[0]
	if math.Abs(location.Latitude-37.775) > 0.000001 || math.Abs(location.Longitude-(-122.4166666667)) > 0.000001 {
		t.Fatalf("unexpected location: %+v", location)
	}
	if len(result.Metadata.Comments) != 2 {
		t.Fatalf("unexpected comments: %+v", result.Metadata.Comments)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "does not match") {
		t.Fatalf("expected one extension mismatch warning, got %v", result.Warnings)
	}
}

func TestRejectsUnsupportedBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.jpg")
	if err := os.WriteFile(path, []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := File(path); err == nil || !strings.Contains(err.Error(), "unsupported file content") {
		t.Fatalf("expected unsupported-content error, got %v", err)
	}
}

func encodePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	imageData := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			imageData.Set(x, y, color.NRGBA{R: 32, G: 64, B: 96, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, imageData); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func insertBeforeIEND(base []byte, chunks ...[]byte) []byte {
	result := append([]byte{}, base[:len(base)-12]...)
	for _, chunk := range chunks {
		result = append(result, chunk...)
	}
	return append(result, base[len(base)-12:]...)
}

func pngChunk(chunkType string, data []byte) []byte {
	result := make([]byte, 12+len(data))
	binary.BigEndian.PutUint32(result[0:4], uint32(len(data)))
	copy(result[4:8], chunkType)
	copy(result[8:8+len(data)], data)
	checksum := crc32.NewIEEE()
	_, _ = checksum.Write(result[4 : 8+len(data)])
	binary.BigEndian.PutUint32(result[8+len(data):], checksum.Sum32())
	return result
}

func compressedPNGText(t *testing.T, keyword, value string) []byte {
	t.Helper()
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write([]byte(value)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	result := append([]byte(keyword), 0, 0)
	return append(result, compressed.Bytes()...)
}

func internationalPNGText(keyword string, value []byte) []byte {
	result := append([]byte(keyword), 0, 0, 0, 0, 0)
	return append(result, value...)
}

func jpegSegment(marker byte, payload []byte) []byte {
	result := []byte{0xff, marker, 0, 0}
	binary.BigEndian.PutUint16(result[2:4], uint16(len(payload)+2))
	return append(result, payload...)
}

func testTIFF() []byte {
	data := make([]byte, 320)
	copy(data[0:2], "II")
	binary.LittleEndian.PutUint16(data[2:4], 42)
	binary.LittleEndian.PutUint32(data[4:8], 8)

	binary.LittleEndian.PutUint16(data[8:10], 4)
	putTIFFEntry(data[10:22], 0x010f, 2, 10, 64)
	putTIFFEntry(data[22:34], 0x0110, 2, 10, 74)
	putTIFFEntry(data[34:46], 0x8769, 4, 1, 100)
	putTIFFEntry(data[46:58], 0x8825, 4, 1, 140)
	copy(data[64:74], "Acme Corp\x00")
	copy(data[74:84], "Model One\x00")

	binary.LittleEndian.PutUint16(data[100:102], 2)
	putTIFFEntry(data[102:114], 0x9003, 2, 20, 200)
	putTIFFEntry(data[114:126], 0x9286, 7, 20, 220)
	copy(data[200:220], "2026:07:18 12:34:56\x00")
	copy(data[220:240], "ASCII\x00\x00\x00Private note")

	binary.LittleEndian.PutUint16(data[140:142], 4)
	putTIFFInlineASCII(data[142:154], 0x0001, "N")
	putTIFFEntry(data[154:166], 0x0002, 5, 3, 250)
	putTIFFInlineASCII(data[166:178], 0x0003, "W")
	putTIFFEntry(data[178:190], 0x0004, 5, 3, 274)
	putRationals(data[250:274], [][2]uint32{{37, 1}, {46, 1}, {30, 1}})
	putRationals(data[274:298], [][2]uint32{{122, 1}, {25, 1}, {0, 1}})
	return data
}

func putTIFFEntry(target []byte, tag, valueType uint16, count, value uint32) {
	binary.LittleEndian.PutUint16(target[0:2], tag)
	binary.LittleEndian.PutUint16(target[2:4], valueType)
	binary.LittleEndian.PutUint32(target[4:8], count)
	binary.LittleEndian.PutUint32(target[8:12], value)
}

func putTIFFInlineASCII(target []byte, tag uint16, value string) {
	putTIFFEntry(target, tag, 2, 2, 0)
	target[8] = value[0]
	target[9] = 0
}

func putRationals(target []byte, values [][2]uint32) {
	for index, value := range values {
		binary.LittleEndian.PutUint32(target[index*8:index*8+4], value[0])
		binary.LittleEndian.PutUint32(target[index*8+4:index*8+8], value[1])
	}
}
