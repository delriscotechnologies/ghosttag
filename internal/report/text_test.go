package report

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/delriscotechnologies/ghosttag/internal/assessment"
	"github.com/delriscotechnologies/ghosttag/internal/model"
)

func TestWriteTextUsesContextWithoutSeverity(t *testing.T) {
	metadata := model.Metadata{
		Locations:   []model.Location{{Latitude: 37.775, Longitude: -122.416667, Source: "EXIF"}},
		CaptureTime: []model.SourcedValue{{Value: "2026-07-18T12:34:56 (timezone not recorded)", Source: "EXIF"}},
		DeviceModel: []model.SourcedValue{{Value: "Model One", Source: "EXIF"}},
	}
	result := model.Report{
		File: model.FileInfo{
			Name: "sample.jpg", Extension: ".jpg", DetectedFormat: "JPEG",
			Size: 2048, SHA256: strings.Repeat("a", 64), Width: 3, Height: 2,
		},
		Metadata:   metadata,
		Assessment: assessment.Evaluate(metadata),
	}

	var output bytes.Buffer
	if err := WriteText(&output, result); err != nil {
		t.Fatalf("WriteText returned an error: %v", err)
	}
	text := output.String()
	for _, expected := range []string{
		"ghosttag — image metadata report",
		"Detected format: JPEG",
		"Categories found (3): location, capture time, device",
		"In combination, these details can reveal more context",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("report is missing %q:\n%s", expected, text)
		}
	}
	for _, forbidden := range []string{"HIGH", "MEDIUM", "LOW", "risk score", "severity"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("report contains forbidden severity wording %q:\n%s", forbidden, text)
		}
	}
}

func TestWriteTextReturnsLateWriterErrors(t *testing.T) {
	writer := &failAfterWriter{remaining: 40}
	result := model.Report{File: model.FileInfo{Name: "sample.png", DetectedFormat: "PNG"}}
	if err := WriteText(writer, result); !errors.Is(err, errWriterFailed) {
		t.Fatalf("WriteText error = %v, want %v", err, errWriterFailed)
	}
}

var errWriterFailed = errors.New("writer failed")

type failAfterWriter struct {
	remaining int
}

func (writer *failAfterWriter) Write(data []byte) (int, error) {
	if writer.remaining <= 0 {
		return 0, errWriterFailed
	}
	if len(data) > writer.remaining {
		written := writer.remaining
		writer.remaining = 0
		return written, errWriterFailed
	}
	writer.remaining -= len(data)
	return len(data), nil
}
