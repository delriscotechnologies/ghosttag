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
