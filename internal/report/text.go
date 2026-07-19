package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/delriscotechnologies/ghosttag/internal/model"
)

type textWriter struct {
	writer io.Writer
	err    error
}

func (writer *textWriter) print(format string, args ...any) {
	if writer.err != nil {
		return
	}
	_, writer.err = fmt.Fprintf(writer.writer, format, args...)
}

func (writer *textWriter) println(args ...any) {
	if writer.err != nil {
		return
	}
	_, writer.err = fmt.Fprintln(writer.writer, args...)
}

// WriteText writes a stable, human-readable terminal report.
func WriteText(output io.Writer, result model.Report) error {
	writer := &textWriter{writer: output}
	writer.println("ghosttag — image metadata report")

	writer.println("\nFile")
	writer.print("  Name: %s\n", result.File.Name)
	writer.print("  Detected format: %s\n", result.File.DetectedFormat)
	writer.print("  Extension: %s\n", displayExtension(result.File.Extension))
	writer.print("  Size: %s (%d bytes)\n", humanSize(result.File.Size), result.File.Size)
	if result.File.Width > 0 && result.File.Height > 0 {
		writer.print("  Dimensions: %d × %d pixels\n", result.File.Width, result.File.Height)
	}
	writer.print("  SHA-256: %s\n", result.File.SHA256)

	writer.println("\nMetadata")
	if len(result.Metadata.Containers) > 0 {
		writer.print("  Containers: %s\n", strings.Join(result.Metadata.Containers, ", "))
	} else {
		writer.println("  Containers: none found")
	}

	if !result.Metadata.HasSupportedFields() {
		writer.println("  No supported metadata fields were found.")
	} else {
		writeLocations(writer, "Location", result.Metadata.Locations)
		writeValues(writer, "Capture time", result.Metadata.CaptureTime)
		writeValues(writer, "Device make", result.Metadata.DeviceMake)
		writeValues(writer, "Device model", result.Metadata.DeviceModel)
		writeValues(writer, "Software", result.Metadata.Software)
		writeValues(writer, "Author", result.Metadata.Authors)
		writeValues(writer, "Copyright", result.Metadata.Copyright)
		writeValues(writer, "Comment or description", result.Metadata.Comments)
		writeValues(writer, "Orientation", result.Metadata.Orientation)
	}

	if len(result.Warnings) > 0 {
		writer.println("\nWarnings")
		for _, warning := range result.Warnings {
			writer.print("  - %s\n", warning)
		}
	}

	writer.println("\nPrivacy context")
	switch len(result.Assessment.Categories) {
	case 0:
		writer.println("  No privacy-relevant metadata categories were found.")
		writer.println("  This does not prove the image is anonymous; ghosttag only inspects supported metadata.")
	default:
		writer.print("  Categories found (%d): %s\n", len(result.Assessment.Categories), strings.Join(result.Assessment.Categories, ", "))
		if result.Assessment.Notice != "" {
			writer.print("  Note: %s\n", result.Assessment.Notice)
		}
	}

	return writer.err
}

func writeValues(writer *textWriter, label string, values []model.SourcedValue) {
	if len(values) == 0 {
		return
	}
	writer.print("  %s:\n", label)
	for _, value := range values {
		writer.print("    - %s [%s]\n", value.Value, value.Source)
	}
}

func writeLocations(writer *textWriter, label string, locations []model.Location) {
	if len(locations) == 0 {
		return
	}
	writer.print("  %s:\n", label)
	for _, location := range locations {
		writer.print("    - %.6f, %.6f [%s]\n", location.Latitude, location.Longitude, location.Source)
	}
}

func displayExtension(extension string) string {
	if extension == "" {
		return "none"
	}
	return extension
}

func humanSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	divisor, exponent := int64(unit), 0
	for value := size / unit; value >= unit && exponent < 3; value /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(divisor), "KMGT"[exponent])
}
