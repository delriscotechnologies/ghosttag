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

func TestSafeTextRemovesFormatControls(t *testing.T) {
	got := safeText("safe\u202Egpj.exe\u2066end")
	if strings.ContainsRune(got, '\u202E') || strings.ContainsRune(got, '\u2066') {
		t.Fatalf("format control survived: %q", got)
	}
}

func TestCollectorLimitsValues(t *testing.T) {
	var metadata model.Metadata
	var warnings []string
	collector := newCollector(&metadata, &warnings)
	for index := 0; index < maximumMetadataValues+20; index++ {
		collector.addValue(&metadata.Comments, string(rune(0x1000+index)), "test")
	}
	if len(metadata.Comments) != maximumMetadataValues {
		t.Fatalf("got %d values", len(metadata.Comments))
	}
	if len(warnings) == 0 {
		t.Fatal("expected limit warning")
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
