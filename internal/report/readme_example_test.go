package report

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/delriscotechnologies/ghosttag/internal/assessment"
	"github.com/delriscotechnologies/ghosttag/internal/model"
)

func TestREADMEExampleMatchesReportOutput(t *testing.T) {
	metadata := model.Metadata{
		Containers: []string{"PNG tEXt", "PNG iTXt", "PNG XMP"},
		CaptureTime: []model.SourcedValue{{
			Value: "2026-07-18T12:34:56Z", Source: "PNG XMP",
		}},
		Authors: []model.SourcedValue{{
			Value: "Example Author", Source: "PNG tEXt:Author",
		}},
		Comments: []model.SourcedValue{{
			Value: "Example comment", Source: "PNG tEXt:Comment",
		}},
	}
	result := model.Report{
		File: model.FileInfo{
			Name:           "sample-metadata.png",
			DetectedFormat: "PNG",
			Extension:      ".png",
			Size:           402,
			Width:          2,
			Height:         2,
			SHA256:         "2f19e2d43cf22ae8ffe544505f01e30e03cffbbd02075ee7e0afcf3bf78fed63",
		},
		Metadata: metadata,
	}
	result.Assessment = assessment.Evaluate(metadata)

	var output bytes.Buffer
	if err := WriteText(&output, result); err != nil {
		t.Fatal(err)
	}

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test source")
	}
	readmePath := filepath.Join(filepath.Dir(sourceFile), "..", "..", "README.md")
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}

	expectedBlock := "```text\n" + output.String() + "```"
	if !strings.Contains(string(readme), expectedBlock) {
		t.Fatalf("README example is out of sync with report output:\n%s", output.String())
	}
}
