package inspect

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"
)

const (
	ifdPrimary = iota
	ifdEXIF
	ifdGPS
)

type tiffParser struct {
	data      []byte
	order     binary.ByteOrder
	source    string
	collector *collector
	visited   map[uint32]bool
}

type tiffEntry struct {
	tag       uint16
	valueType uint16
	count     uint32
	data      []byte
}

func parseEXIF(data []byte, source string, collector *collector) error {
	if len(data) < 8 {
		return fmt.Errorf("TIFF header is incomplete")
	}

	var order binary.ByteOrder
	switch string(data[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return fmt.Errorf("unknown TIFF byte order")
	}
	if order.Uint16(data[2:4]) != 42 {
		return fmt.Errorf("invalid TIFF magic number")
	}

	parser := &tiffParser{
		data: data, order: order, source: source, collector: collector,
		visited: make(map[uint32]bool),
	}
	return parser.parseIFD(order.Uint32(data[4:8]), ifdPrimary)
}

func (parser *tiffParser) parseIFD(offset uint32, kind int) error {
	if parser.visited[offset] {
		return fmt.Errorf("cyclic TIFF directory reference")
	}
	parser.visited[offset] = true

	position := uint64(offset)
	if position+2 > uint64(len(parser.data)) {
		return fmt.Errorf("TIFF directory offset is outside the metadata")
	}
	entryCount := uint64(parser.order.Uint16(parser.data[position : position+2]))
	entriesEnd := position + 2 + entryCount*12
	if entriesEnd+4 > uint64(len(parser.data)) {
		return fmt.Errorf("TIFF directory is incomplete")
	}

	var exifOffset, gpsOffset uint32
	var latitudeRef, longitudeRef string
	var latitude, longitude []float64
	var originalTime, digitizedTime string
	var generalOffset, originalOffset, digitizedOffset string

	for index := uint64(0); index < entryCount; index++ {
		entryPosition := position + 2 + index*12
		entry, err := parser.readEntry(entryPosition)
		if err != nil {
			return err
		}

		switch kind {
		case ifdPrimary:
			switch entry.tag {
			case 0x010e:
				parser.collector.addValue(&parser.collector.metadata.Comments, entry.text(), parser.source)
			case 0x010f:
				parser.collector.addValue(&parser.collector.metadata.DeviceMake, entry.text(), parser.source)
			case 0x0110:
				parser.collector.addValue(&parser.collector.metadata.DeviceModel, entry.text(), parser.source)
			case 0x0112:
				if value, ok := parser.entryUnsigned(entry); ok {
					parser.collector.addValue(&parser.collector.metadata.Orientation, orientationName(value), parser.source)
				}
			case 0x0131:
				parser.collector.addValue(&parser.collector.metadata.Software, entry.text(), parser.source)
			case 0x013b:
				parser.collector.addValue(&parser.collector.metadata.Authors, entry.text(), parser.source)
			case 0x8298:
				parser.collector.addValue(&parser.collector.metadata.Copyright, entry.text(), parser.source)
			case 0x8769:
				exifOffset, _ = parser.entryUnsigned(entry)
			case 0x8825:
				gpsOffset, _ = parser.entryUnsigned(entry)
			}
		case ifdEXIF:
			switch entry.tag {
			case 0x9003:
				originalTime = entry.text()
			case 0x9004:
				digitizedTime = entry.text()
			case 0x9010:
				generalOffset = entry.text()
			case 0x9011:
				originalOffset = entry.text()
			case 0x9012:
				digitizedOffset = entry.text()
			case 0x9286:
				parser.collector.addValue(&parser.collector.metadata.Comments, decodeUserComment(entry.data, parser.order), parser.source)
			case 0xa430:
				parser.collector.addValue(&parser.collector.metadata.Authors, entry.text(), parser.source)
			}
		case ifdGPS:
			switch entry.tag {
			case 0x0001:
				latitudeRef = strings.ToUpper(entry.text())
			case 0x0002:
				latitude, err = parser.entryRationals(entry)
				if err != nil {
					return fmt.Errorf("invalid GPS latitude: %w", err)
				}
			case 0x0003:
				longitudeRef = strings.ToUpper(entry.text())
			case 0x0004:
				longitude, err = parser.entryRationals(entry)
				if err != nil {
					return fmt.Errorf("invalid GPS longitude: %w", err)
				}
			}
		}
	}

	if kind == ifdPrimary {
		if exifOffset != 0 {
			if err := parser.parseIFD(exifOffset, ifdEXIF); err != nil {
				return fmt.Errorf("EXIF subdirectory: %w", err)
			}
		}
		if gpsOffset != 0 {
			if err := parser.parseIFD(gpsOffset, ifdGPS); err != nil {
				return fmt.Errorf("GPS subdirectory: %w", err)
			}
		}
	}
	if kind == ifdEXIF {
		if originalOffset == "" {
			originalOffset = generalOffset
		}
		if digitizedOffset == "" {
			digitizedOffset = generalOffset
		}
		if originalTime != "" {
			parser.collector.addCaptureTime(withEXIFOffset(originalTime, originalOffset), parser.source)
		}
		if digitizedTime != "" {
			parser.collector.addCaptureTime(withEXIFOffset(digitizedTime, digitizedOffset), parser.source)
		}
	}

	if kind == ifdGPS && (len(latitude) > 0 || len(longitude) > 0) {
		if latitudeRef != "N" && latitudeRef != "S" || longitudeRef != "E" && longitudeRef != "W" {
			parser.collector.warn("Ignored GPS coordinates from %s because direction references were missing or invalid.", parser.source)
			return nil
		}
		if len(latitude) != 3 || len(longitude) != 3 {
			parser.collector.warn("Ignored GPS coordinates from %s because degrees, minutes, and seconds were incomplete.", parser.source)
			return nil
		}
		lat, err := coordinateFromDMS(latitude, 90)
		if err != nil {
			parser.collector.warn("Ignored invalid GPS latitude from %s: %v.", parser.source, err)
			return nil
		}
		lon, err := coordinateFromDMS(longitude, 180)
		if err != nil {
			parser.collector.warn("Ignored invalid GPS longitude from %s: %v.", parser.source, err)
			return nil
		}
		if latitudeRef == "S" {
			lat = -lat
		}
		if longitudeRef == "W" {
			lon = -lon
		}
		parser.collector.addLocation(lat, lon, parser.source)
	}

	return nil
}

func withEXIFOffset(value, offset string) string {
	value = safeText(value)
	offset = safeText(offset)
	if len(value) != 19 || len(offset) != 6 ||
		(offset[0] != '+' && offset[0] != '-') || offset[3] != ':' {
		return value
	}
	for _, position := range []int{1, 2, 4, 5} {
		if offset[position] < '0' || offset[position] > '9' {
			return value
		}
	}
	return value[0:4] + "-" + value[5:7] + "-" + value[8:10] + "T" + value[11:] + offset
}

func (parser *tiffParser) readEntry(position uint64) (tiffEntry, error) {
	if position+12 > uint64(len(parser.data)) {
		return tiffEntry{}, fmt.Errorf("TIFF entry is incomplete")
	}
	entryBytes := parser.data[position : position+12]
	entry := tiffEntry{
		tag:       parser.order.Uint16(entryBytes[0:2]),
		valueType: parser.order.Uint16(entryBytes[2:4]),
		count:     parser.order.Uint32(entryBytes[4:8]),
	}

	typeSizes := map[uint16]uint64{
		1: 1, 2: 1, 3: 2, 4: 4, 5: 8,
		6: 1, 7: 1, 8: 2, 9: 4, 10: 8,
		11: 4, 12: 8,
	}
	typeSize, known := typeSizes[entry.valueType]
	if !known {
		return tiffEntry{}, fmt.Errorf("unsupported TIFF field type %d", entry.valueType)
	}
	valueLength := uint64(entry.count) * typeSize
	if entry.count != 0 && valueLength/typeSize != uint64(entry.count) {
		return tiffEntry{}, fmt.Errorf("TIFF field length overflow")
	}

	if valueLength <= 4 {
		entry.data = entryBytes[8 : 8+valueLength]
		return entry, nil
	}
	valueOffset := uint64(parser.order.Uint32(entryBytes[8:12]))
	if valueOffset+valueLength > uint64(len(parser.data)) {
		return tiffEntry{}, fmt.Errorf("TIFF field points outside the metadata")
	}
	entry.data = parser.data[valueOffset : valueOffset+valueLength]
	return entry, nil
}

func (entry tiffEntry) text() string {
	return string(bytes.Trim(entry.data, "\x00"))
}

func (parser *tiffParser) entryUnsigned(entry tiffEntry) (uint32, bool) {
	switch entry.valueType {
	case 1, 7:
		if len(entry.data) >= 1 {
			return uint32(entry.data[0]), true
		}
	case 3:
		if len(entry.data) >= 2 {
			return uint32(parser.order.Uint16(entry.data[:2])), true
		}
	case 4:
		if len(entry.data) >= 4 {
			return parser.order.Uint32(entry.data[:4]), true
		}
	}
	return 0, false
}

func (parser *tiffParser) entryRationals(entry tiffEntry) ([]float64, error) {
	if entry.valueType != 5 && entry.valueType != 10 {
		return nil, fmt.Errorf("expected a rational field")
	}
	if len(entry.data)%8 != 0 {
		return nil, fmt.Errorf("incomplete rational value")
	}

	values := make([]float64, 0, len(entry.data)/8)
	for position := 0; position < len(entry.data); position += 8 {
		if entry.valueType == 5 {
			numerator := parser.order.Uint32(entry.data[position : position+4])
			denominator := parser.order.Uint32(entry.data[position+4 : position+8])
			if denominator == 0 {
				return nil, fmt.Errorf("zero rational denominator")
			}
			values = append(values, float64(numerator)/float64(denominator))
			continue
		}

		numerator := int32(parser.order.Uint32(entry.data[position : position+4]))
		denominator := int32(parser.order.Uint32(entry.data[position+4 : position+8]))
		if denominator == 0 {
			return nil, fmt.Errorf("zero rational denominator")
		}
		values = append(values, float64(numerator)/float64(denominator))
	}
	return values, nil
}

func orientationName(value uint32) string {
	names := map[uint32]string{
		1: "1 (normal)",
		2: "2 (mirrored horizontally)",
		3: "3 (rotated 180°)",
		4: "4 (mirrored vertically)",
		5: "5 (mirrored horizontally, then rotated 270° clockwise)",
		6: "6 (rotated 90° clockwise)",
		7: "7 (mirrored horizontally, then rotated 90° clockwise)",
		8: "8 (rotated 270° clockwise)",
	}
	if name, ok := names[value]; ok {
		return name
	}
	return fmt.Sprintf("%d (unknown)", value)
}

func decodeUserComment(data []byte, order binary.ByteOrder) string {
	if len(data) < 8 {
		return string(data)
	}
	prefix := string(data[:8])
	payload := bytes.Trim(data[8:], "\x00")
	switch {
	case strings.HasPrefix(prefix, "ASCII"):
		return string(payload)
	case strings.HasPrefix(prefix, "UNICODE"):
		return decodeUTF16(payload, order)
	default:
		return string(bytes.Trim(data, "\x00"))
	}
}

func decodeUTF16(data []byte, order binary.ByteOrder) string {
	if len(data) < 2 {
		return string(data)
	}
	if data[0] == 0xfe && data[1] == 0xff {
		order = binary.BigEndian
		data = data[2:]
	} else if data[0] == 0xff && data[1] == 0xfe {
		order = binary.LittleEndian
		data = data[2:]
	}
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	values := make([]uint16, 0, len(data)/2)
	for position := 0; position < len(data); position += 2 {
		values = append(values, order.Uint16(data[position:position+2]))
	}
	return strings.TrimRight(string(utf16.Decode(values)), "\x00")
}
