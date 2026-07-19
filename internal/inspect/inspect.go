package inspect

import (
	"crypto/sha256"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/delriscotechnologies/ghosttag/internal/model"
)

const maximumFileBytes int64 = 100 * 1024 * 1024

var pngSignature = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}

// File inspects one JPEG or PNG without changing it.
func File(path string) (model.Report, error) {
	var result model.Report

	file, err := openForInspection(path)
	if err != nil {
		return result, fmt.Errorf("open %q: %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return result, fmt.Errorf("stat %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return result, fmt.Errorf("%q is not a regular file; expected one JPEG or PNG file", path)
	}
	if info.Size() > maximumFileBytes {
		return result, fmt.Errorf("%q exceeds the %d MiB safety limit", path, maximumFileBytes/(1024*1024))
	}

	data, err := io.ReadAll(io.LimitReader(file, maximumFileBytes+1))
	if err != nil {
		return result, fmt.Errorf("read %q: %w", path, err)
	}
	if int64(len(data)) > maximumFileBytes {
		return result, fmt.Errorf("%q exceeds the %d MiB safety limit", path, maximumFileBytes/(1024*1024))
	}

	format, err := detectFormat(data)
	if err != nil {
		return result, err
	}

	digest := sha256.Sum256(data)
	extension := strings.ToLower(filepath.Ext(path))
	result.File = model.FileInfo{
		Name:           safeText(filepath.Base(path)),
		Extension:      extension,
		DetectedFormat: format,
		Size:           int64(len(data)),
		SHA256:         fmt.Sprintf("%x", digest),
	}

	collector := newCollector(&result.Metadata, &result.Warnings)
	switch format {
	case "JPEG":
		result.File.Width, result.File.Height, err = parseJPEG(data, collector)
	case "PNG":
		result.File.Width, result.File.Height, err = parsePNG(data, collector)
	}
	if err != nil {
		return model.Report{}, err
	}

	if !extensionMatchesFormat(extension, format) {
		result.Warnings = appendUnique(result.Warnings, fmt.Sprintf(
			"The %s extension does not match the detected %s format.",
			displayExtension(extension), format,
		))
	}

	return result, nil
}

func detectFormat(data []byte) (string, error) {
	if len(data) >= len(pngSignature) && string(data[:len(pngSignature)]) == string(pngSignature) {
		return "PNG", nil
	}
	if len(data) >= 2 && data[0] == 0xff && data[1] == 0xd8 {
		return "JPEG", nil
	}
	return "", fmt.Errorf("unsupported file content; expected JPEG or PNG bytes")
}

func extensionMatchesFormat(extension, format string) bool {
	switch format {
	case "JPEG":
		return extension == ".jpg" || extension == ".jpeg"
	case "PNG":
		return extension == ".png"
	default:
		return false
	}
}

func displayExtension(extension string) string {
	if extension == "" {
		return "missing"
	}
	return extension
}
