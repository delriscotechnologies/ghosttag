package inspect

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	jpegAPP1 = 0xe1
	jpegCOM  = 0xfe
	jpegSOS  = 0xda
	jpegEOI  = 0xd9
)

var (
	exifHeader = []byte{'E', 'x', 'i', 'f', 0, 0}
	xmpHeader  = []byte("http://ns.adobe.com/xap/1.0/\x00")
)

func parseJPEG(data []byte, collector *collector) (int, int, error) {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return 0, 0, fmt.Errorf("malformed JPEG: missing start-of-image marker")
	}

	width, height := 0, 0
	for position := 2; position < len(data); {
		if data[position] != 0xff {
			return 0, 0, fmt.Errorf("malformed JPEG: expected a marker at byte %d", position)
		}
		for position < len(data) && data[position] == 0xff {
			position++
		}
		if position >= len(data) {
			return 0, 0, fmt.Errorf("malformed JPEG: incomplete marker")
		}

		marker := data[position]
		position++
		if marker == jpegSOS || marker == jpegEOI {
			break
		}
		if marker == 0x01 || marker >= 0xd0 && marker <= 0xd7 {
			continue
		}
		if position+2 > len(data) {
			return 0, 0, fmt.Errorf("malformed JPEG: incomplete marker length")
		}

		segmentLength := int(binary.BigEndian.Uint16(data[position : position+2]))
		if segmentLength < 2 || position+segmentLength > len(data) {
			return 0, 0, fmt.Errorf("malformed JPEG: invalid marker length")
		}
		payload := data[position+2 : position+segmentLength]
		position += segmentLength

		switch {
		case marker == jpegAPP1 && hasPrefix(payload, exifHeader):
			collector.addContainer("JPEG APP1 EXIF")
			if err := parseEXIF(payload[len(exifHeader):], "JPEG APP1 EXIF", collector); err != nil {
				collector.warn("Could not parse JPEG EXIF metadata: %v", err)
			}
		case marker == jpegAPP1 && hasPrefix(payload, xmpHeader):
			collector.addContainer("JPEG APP1 XMP")
			if err := parseXMP(payload[len(xmpHeader):], "JPEG APP1 XMP", collector); err != nil {
				collector.warn("Could not parse JPEG XMP metadata: %v", err)
			}
		case marker == jpegCOM:
			collector.addContainer("JPEG COM")
			collector.addValue(&collector.metadata.Comments, string(payload), "JPEG COM")
		case isStartOfFrame(marker) && len(payload) >= 5:
			height = int(binary.BigEndian.Uint16(payload[1:3]))
			width = int(binary.BigEndian.Uint16(payload[3:5]))
		}
	}

	return width, height, nil
}

func isStartOfFrame(marker byte) bool {
	if marker < 0xc0 || marker > 0xcf {
		return false
	}
	switch marker {
	case 0xc4, 0xc8, 0xcc:
		return false
	default:
		return true
	}
}

func hasPrefix(data, prefix []byte) bool {
	return len(data) >= len(prefix) && strings.EqualFold(string(data[:len(prefix)]), string(prefix))
}
