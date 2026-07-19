package inspect

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	jpegAPP1 = 0xe1
	jpegCOM  = 0xfe
	jpegSOS  = 0xda
	jpegEOI  = 0xd9

	maximumJPEGMarkers = 100000
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
	foundEnd := false
	markerCount := 0
	for position := 2; position < len(data); {
		markerCount++
		if markerCount > maximumJPEGMarkers {
			return 0, 0, fmt.Errorf("malformed JPEG: exceeds the %d-marker safety limit", maximumJPEGMarkers)
		}
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
		if marker == 0x00 {
			return 0, 0, fmt.Errorf("malformed JPEG: unexpected stuffed byte outside scan data")
		}
		if marker == jpegEOI {
			foundEnd = true
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
		case isStartOfFrame(marker):
			frameWidth, frameHeight, err := parseJPEGFrameHeader(payload)
			if err != nil {
				return 0, 0, err
			}
			width, height = frameWidth, frameHeight
		}

		if marker == jpegSOS {
			var err error
			position, err = skipJPEGScan(data, position)
			if err != nil {
				return 0, 0, err
			}
		}
	}

	if !foundEnd {
		return 0, 0, fmt.Errorf("malformed JPEG: missing end-of-image marker")
	}
	if width == 0 || height == 0 {
		return 0, 0, fmt.Errorf("malformed JPEG: missing or invalid start-of-frame dimensions")
	}
	return width, height, nil
}

func parseJPEGFrameHeader(payload []byte) (int, int, error) {
	if len(payload) < 6 {
		return 0, 0, fmt.Errorf("malformed JPEG: incomplete start-of-frame header")
	}
	precision := payload[0]
	height := int(binary.BigEndian.Uint16(payload[1:3]))
	width := int(binary.BigEndian.Uint16(payload[3:5]))
	components := int(payload[5])
	if precision == 0 || width == 0 || height == 0 || components == 0 {
		return 0, 0, fmt.Errorf("malformed JPEG: invalid start-of-frame values")
	}
	if len(payload) != 6+3*components {
		return 0, 0, fmt.Errorf("malformed JPEG: incomplete start-of-frame component table")
	}
	return width, height, nil
}

func skipJPEGScan(data []byte, position int) (int, error) {
	for position < len(data) {
		relative := bytes.IndexByte(data[position:], 0xff)
		if relative < 0 {
			return 0, fmt.Errorf("malformed JPEG: scan data reaches end of file")
		}
		markerStart := position + relative
		next := markerStart + 1
		for next < len(data) && data[next] == 0xff {
			next++
		}
		if next >= len(data) {
			return 0, fmt.Errorf("malformed JPEG: incomplete marker after scan data")
		}
		marker := data[next]
		if marker == 0x00 || marker >= 0xd0 && marker <= 0xd7 {
			position = next + 1
			continue
		}
		return markerStart, nil
	}
	return 0, fmt.Errorf("malformed JPEG: scan data reaches end of file")
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
