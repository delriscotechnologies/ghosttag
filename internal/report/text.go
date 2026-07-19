package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/delriscotechnologies/ghosttag/internal/model"
)

// WriteText writes a stable, human-readable terminal report.
func WriteText(writer io.Writer, result model.Report) error {
	if _, err := fmt.Fprintln(writer, "ghosttag — image metadata report"); err != nil {
		return err
	}

	fmt.Fprintln(writer, "\nFile")
	fmt.Fprintf(writer, "  Name: %s\n", result.File.Name)
	fmt.Fprintf(writer, "  Detected format: %s\n", result.File.DetectedFormat)
	fmt.Fprintf(writer, "  Extension: %s\n", displayExtension(result.File.Extension))
	fmt.Fprintf(writer, "  Size: %s (%d bytes)\n", humanSize(result.File.Size), result.File.Size)
	if result.File.Width > 0 && result.File.Height > 0 {
		fmt.Fprintf(writer, "  Dimensions: %d × %d pixels\n", result.File.Width, result.File.Height)
	}
	fmt.Fprintf(writer, "  SHA-256: %s\n", result.File.SHA256)

	fmt.Fprintln(writer, "\nMetadata")
	if len(result.Metadata.Containers) > 0 {
		fmt.Fprintf(writer, "  Containers: %s\n", strings.Join(result.Metadata.Containers, ", "))
	} else {
		fmt.Fprintln(writer, "  Containers: none found")
	}

	if !result.Metadata.HasSupportedFields() {
		fmt.Fprintln(writer, "  No supported metadata fields were found.")
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
		fmt.Fprintln(writer, "\nWarnings")
		for _, warning := range result.Warnings {
			fmt.Fprintf(writer, "  - %s\n", warning)
		}
	}

	fmt.Fprintln(writer, "\nPrivacy context")
	switch len(result.Assessment.Categories) {
	case 0:
		fmt.Fprintln(writer, "  No privacy-relevant metadata categories were found.")
		fmt.Fprintln(writer, "  This does not prove the image is anonymous; ghosttag only inspects supported metadata.")
	default:
		fmt.Fprintf(writer, "  Categories found (%d): %s\n", len(result.Assessment.Categories), strings.Join(result.Assessment.Categories, ", "))
		if result.Assessment.Notice != "" {
			fmt.Fprintf(writer, "  Note: %s\n", result.Assessment.Notice)
		}
	}

	return nil
}

func writeValues(writer io.Writer, label string, values []model.SourcedValue) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(writer, "  %s:\n", label)
	for _, value := range values {
		fmt.Fprintf(writer, "    - %s [%s]\n", value.Value, value.Source)
	}
}

func writeLocations(writer io.Writer, label string, locations []model.Location) {
	if len(locations) == 0 {
		return
	}
	fmt.Fprintf(writer, "  %s:\n", label)
	for _, location := range locations {
		fmt.Fprintf(writer, "    - %.6f, %.6f [%s]\n", location.Latitude, location.Longitude, location.Source)
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
