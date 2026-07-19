package inspect

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/delriscotechnologies/ghosttag/internal/model"
)

func TestRejectsOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maximumFileBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := File(path); err == nil || !strings.Contains(err.Error(), "safety limit") {
		t.Fatalf("expected limit error, got %v", err)
	}
}

func TestRejectsSymlink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link.png")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if _, err := File(link); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected regular-file error, got %v", err)
	}
}

func TestRejectsNonFiniteCoordinates(t *testing.T) {
	for _, value := range []string{"NaN", "Inf", "-Inf"} {
		if _, err := parseXMPCoordinate(value, true); err == nil {
			t.Fatalf("%q: expected an error", value)
		}
	}
}

func TestRejectsInvalidXMPDMSCoordinates(t *testing.T) {
	tests := []struct {
		value    string
		latitude bool
	}{
		{value: "37,60N", latitude: true},
		{value: "37,10,60N", latitude: true},
		{value: "37E", latitude: true},
		{value: "122N", latitude: false},
		{value: "90,0.1N", latitude: true},
		{value: "-37N", latitude: true},
	}
	for _, test := range tests {
		if _, err := parseXMPCoordinate(test.value, test.latitude); err == nil {
			t.Errorf("%q: expected an error", test.value)
		}
	}
}

func TestRejectsInvalidEXIFDMSComponents(t *testing.T) {
	data := testTIFF()
	putRationals(data[250:274], [][2]uint32{{37, 1}, {60, 1}, {0, 1}})
	var metadata model.Metadata
	var warnings []string
	if err := parseEXIF(data, "test", newCollector(&metadata, &warnings)); err != nil {
		t.Fatal(err)
	}
	if len(metadata.Locations) != 0 || len(warnings) == 0 {
		t.Fatalf("expected invalid location to be omitted with a warning: locations=%v warnings=%v", metadata.Locations, warnings)
	}
}

func TestXMPRequiresKnownNamespacesAndKeepsLocationsSeparate(t *testing.T) {
	xmp := `<x:xmpmeta xmlns:x="adobe:ns:meta/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:exif="http://ns.adobe.com/exif/1.0/" xmlns:fake="https://example.invalid/fake"><rdf:RDF><rdf:Description dc:creator="Valid Author" fake:creator="Fake Author" exif:GPSLatitude="1N" exif:GPSLongitude="2E"/><rdf:Description exif:GPSLatitude="3N" exif:GPSLongitude="4E"/></rdf:RDF></x:xmpmeta>`
	var metadata model.Metadata
	var warnings []string
	if err := parseXMP([]byte(xmp), "XMP", newCollector(&metadata, &warnings)); err != nil {
		t.Fatal(err)
	}
	if len(metadata.Authors) != 1 || metadata.Authors[0].Value != "Valid Author" {
		t.Fatalf("unexpected authors: %+v", metadata.Authors)
	}
	if len(metadata.Locations) != 2 || metadata.Locations[0].Latitude != 1 || metadata.Locations[0].Longitude != 2 || metadata.Locations[1].Latitude != 3 || metadata.Locations[1].Longitude != 4 {
		t.Fatalf("unexpected locations: %+v", metadata.Locations)
	}
}

func TestXMPDoesNotCrossPairIncompleteDescriptions(t *testing.T) {
	xmp := `<x:xmpmeta xmlns:x="adobe:ns:meta/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:exif="http://ns.adobe.com/exif/1.0/"><rdf:RDF><rdf:Description exif:GPSLatitude="1N"/><rdf:Description exif:GPSLongitude="2E"/></rdf:RDF></x:xmpmeta>`
	var metadata model.Metadata
	var warnings []string
	if err := parseXMP([]byte(xmp), "XMP", newCollector(&metadata, &warnings)); err != nil {
		t.Fatal(err)
	}
	if len(metadata.Locations) != 0 || len(warnings) != 2 {
		t.Fatalf("incomplete descriptions were cross-paired: locations=%v warnings=%v", metadata.Locations, warnings)
	}
}

func TestInvalidCaptureTimeIsNotClassified(t *testing.T) {
	xmp := `<rdf:Description xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:xmp="http://ns.adobe.com/xap/1.0/" xmp:CreateDate="not-a-date"/>`
	var metadata model.Metadata
	var warnings []string
	if err := parseXMP([]byte(xmp), "XMP", newCollector(&metadata, &warnings)); err != nil {
		t.Fatal(err)
	}
	if len(metadata.CaptureTime) != 0 || len(warnings) != 1 {
		t.Fatalf("invalid time was retained: capture=%v warnings=%v", metadata.CaptureTime, warnings)
	}
}

func TestSafeTextRemovesFormatControls(t *testing.T) {
	got := safeText("safe\u202Egpj.exe\u2066end")
	if strings.ContainsRune(got, '\u202E') || strings.ContainsRune(got, '\u2066') {
		t.Fatalf("format control survived: %q", got)
	}
}

func TestSanitizesExtensionAndMismatchWarning(t *testing.T) {
	data := append([]byte{}, pngSignature...)
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], 1)
	binary.BigEndian.PutUint32(ihdr[4:8], 1)
	data = append(data, testChunk("IHDR", ihdr)...)
	data = append(data, testChunk("IEND", nil)...)

	path := filepath.Join(t.TempDir(), "image.\x1b[31mjpg\u202E")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Skipf("platform does not permit the unsafe test filename: %v", err)
	}
	result, err := File(path)
	if err != nil {
		t.Fatal(err)
	}
	values := append([]string{result.File.Name, result.File.Extension}, result.Warnings...)
	for _, value := range values {
		if strings.ContainsRune(value, '\x1b') || strings.ContainsRune(value, '\u202E') {
			t.Fatalf("unsafe terminal character survived in %q", value)
		}
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "does not match") {
		t.Fatalf("expected sanitized extension mismatch warning, got %v", result.Warnings)
	}
}

func TestCollectorLimitsEachFieldIndependently(t *testing.T) {
	var metadata model.Metadata
	var warnings []string
	collector := newCollector(&metadata, &warnings)
	for index := 0; index < maximumMetadataValues+20; index++ {
		collector.addValue(&metadata.Comments, string(rune(0x1000+index)), "comment source")
	}
	collector.addLocation(37.775, -122.416667, "GPS source")
	collector.addCaptureTime("2026-07-18", "time source")

	if len(metadata.Comments) != maximumMetadataValues {
		t.Fatalf("got %d comments", len(metadata.Comments))
	}
	if len(metadata.Locations) != 1 || len(metadata.CaptureTime) != 1 {
		t.Fatalf("one full field hid later privacy categories: %+v", metadata)
	}
	if len(warnings) == 0 {
		t.Fatal("expected limit warning")
	}
}

func TestCollectorRetainsAllSourcesForDuplicateValues(t *testing.T) {
	var metadata model.Metadata
	var warnings []string
	collector := newCollector(&metadata, &warnings)
	collector.addValue(&metadata.Authors, "Example Author", "EXIF")
	collector.addValue(&metadata.Authors, "example author", "XMP")
	collector.addLocation(1, 2, "EXIF GPS")
	collector.addLocation(1, 2, "XMP GPS")

	if len(metadata.Authors) != 1 || metadata.Authors[0].Source != "EXIF | XMP" {
		t.Fatalf("unexpected author provenance: %+v", metadata.Authors)
	}
	if len(metadata.Locations) != 1 || metadata.Locations[0].Source != "EXIF GPS | XMP GPS" {
		t.Fatalf("unexpected location provenance: %+v", metadata.Locations)
	}
}

func TestJPEGRequiresDimensions(t *testing.T) {
	var metadata model.Metadata
	var warnings []string
	_, _, err := parseJPEG([]byte{0xff, 0xd8, 0xff, 0xd9}, newCollector(&metadata, &warnings))
	if err == nil || !strings.Contains(err.Error(), "dimensions") {
		t.Fatalf("expected dimension error, got %v", err)
	}
}

func TestJPEGMarkerLimit(t *testing.T) {
	data := []byte{0xff, 0xd8}
	for index := 0; index <= maximumJPEGMarkers; index++ {
		data = append(data, 0xff, 0x01)
	}
	var metadata model.Metadata
	var warnings []string
	if _, _, err := parseJPEG(data, newCollector(&metadata, &warnings)); err == nil || !strings.Contains(err.Error(), "marker safety limit") {
		t.Fatalf("expected marker limit error, got %v", err)
	}
}

func TestJPEGRejectsIncompleteSOF(t *testing.T) {
	data := []byte{0xff, 0xd8}
	data = append(data, jpegSegment(0xc0, []byte{8, 0, 1, 0, 1})...)
	data = append(data, 0xff, 0xd9)
	var metadata model.Metadata
	var warnings []string
	if _, _, err := parseJPEG(data, newCollector(&metadata, &warnings)); err == nil || !strings.Contains(err.Error(), "start-of-frame") {
		t.Fatalf("expected SOF validation error, got %v", err)
	}
}

func TestJPEGFindsMetadataAfterScan(t *testing.T) {
	data := []byte{0xff, 0xd8}
	data = append(data, jpegSegment(0xc0, []byte{8, 0, 1, 0, 1, 1, 1, 0x11, 0})...)
	data = append(data, jpegSegment(jpegSOS, []byte{1, 1, 0, 0, 63, 0})...)
	data = append(data, 0x11, 0xff, 0x00, 0x22, 0xff, 0xd0, 0x33)
	data = append(data, jpegSegment(jpegCOM, []byte("after scan"))...)
	data = append(data, 0xff, 0xd9)
	var metadata model.Metadata
	var warnings []string
	if _, _, err := parseJPEG(data, newCollector(&metadata, &warnings)); err != nil {
		t.Fatal(err)
	}
	if len(metadata.Comments) != 1 || metadata.Comments[0].Value != "after scan" {
		t.Fatalf("post-scan metadata was not found: %+v", metadata.Comments)
	}
}

func TestPNGChunkLimit(t *testing.T) {
	data := append([]byte{}, pngSignature...)
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], 1)
	binary.BigEndian.PutUint32(ihdr[4:8], 1)
	data = append(data, testChunk("IHDR", ihdr)...)
	for index := 0; index < maximumPNGChunks; index++ {
		data = append(data, testChunk("aaAa", nil)...)
	}
	data = append(data, testChunk("IEND", nil)...)

	var metadata model.Metadata
	var warnings []string
	_, _, err := parsePNG(data, newCollector(&metadata, &warnings))
	if err == nil || !strings.Contains(err.Error(), "chunk safety limit") {
		t.Fatalf("expected chunk limit error, got %v", err)
	}
}

func TestPNGRequiresFirstUniqueIHDRAndValidDimensions(t *testing.T) {
	validIHDR := make([]byte, 13)
	binary.BigEndian.PutUint32(validIHDR[0:4], 1)
	binary.BigEndian.PutUint32(validIHDR[4:8], 1)
	tooWideIHDR := append([]byte{}, validIHDR...)
	binary.BigEndian.PutUint32(tooWideIHDR[0:4], maximumPNGDimension+1)

	tests := [][]byte{
		append(append(append([]byte{}, pngSignature...), testChunk("tEXt", []byte("A\x00B"))...), testChunk("IEND", nil)...),
		append(append(append(append([]byte{}, pngSignature...), testChunk("IHDR", validIHDR)...), testChunk("IHDR", validIHDR)...), testChunk("IEND", nil)...),
		append(append(append([]byte{}, pngSignature...), testChunk("IHDR", tooWideIHDR)...), testChunk("IEND", nil)...),
		append(append(append(append([]byte{}, pngSignature...), testChunk("IHDR", validIHDR)...), testChunk("IEND", nil)...), byte(0)),
	}
	for index, data := range tests {
		var metadata model.Metadata
		var warnings []string
		if _, _, err := parsePNG(data, newCollector(&metadata, &warnings)); err == nil {
			t.Errorf("case %d: expected malformed PNG error", index)
		}
	}
}

func TestPNGRejectsInvalidCriticalCRCAndSkipsInvalidMetadata(t *testing.T) {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], 1)
	binary.BigEndian.PutUint32(ihdr[4:8], 1)
	badIHDR := testChunk("IHDR", ihdr)
	badIHDR[len(badIHDR)-1] ^= 0xff
	critical := append(append(append([]byte{}, pngSignature...), badIHDR...), testChunk("IEND", nil)...)
	var metadata model.Metadata
	var warnings []string
	if _, _, err := parsePNG(critical, newCollector(&metadata, &warnings)); err == nil || !strings.Contains(err.Error(), "invalid CRC") {
		t.Fatalf("expected critical CRC error, got %v", err)
	}

	badText := testChunk("tEXt", []byte("Author\x00Untrusted"))
	badText[len(badText)-1] ^= 0xff
	ancillary := append([]byte{}, pngSignature...)
	ancillary = append(ancillary, testChunk("IHDR", ihdr)...)
	ancillary = append(ancillary, badText...)
	ancillary = append(ancillary, testChunk("IEND", nil)...)
	metadata = model.Metadata{}
	warnings = nil
	if _, _, err := parsePNG(ancillary, newCollector(&metadata, &warnings)); err != nil {
		t.Fatal(err)
	}
	if len(metadata.Authors) != 0 || len(warnings) != 1 {
		t.Fatalf("invalid ancillary chunk was used: authors=%v warnings=%v", metadata.Authors, warnings)
	}
}

func testChunk(kind string, payload []byte) []byte {
	result := make([]byte, 12+len(payload))
	binary.BigEndian.PutUint32(result[:4], uint32(len(payload)))
	copy(result[4:8], kind)
	copy(result[8:], payload)
	checksum := crc32.NewIEEE()
	_, _ = checksum.Write(result[4 : 8+len(payload)])
	binary.BigEndian.PutUint32(result[8+len(payload):], checksum.Sum32())
	return result
}
