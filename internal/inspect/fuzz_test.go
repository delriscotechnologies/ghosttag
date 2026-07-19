package inspect

import (
	"testing"

	"github.com/delriscotechnologies/ghosttag/internal/model"
)

func FuzzImageParsersDoNotPanic(f *testing.F) {
	f.Add([]byte{0xff, 0xd8, 0xff, 0xd9})
	f.Add(append([]byte{}, pngSignature...))
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

func FuzzXMPCoordinatesDoNotPanic(f *testing.F) {
	f.Add("37,46.5N")
	f.Add("NaN")
	f.Fuzz(func(t *testing.T, value string) {
		_, _ = parseXMPCoordinate(value, true)
	})
}
