package inspect

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

const (
	maximumXMPDepth  = 128
	maximumXMPTokens = 100000

	rdfNamespace       = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	dcNamespace        = "http://purl.org/dc/elements/1.1/"
	xmpNamespace       = "http://ns.adobe.com/xap/1.0/"
	tiffNamespace      = "http://ns.adobe.com/tiff/1.0/"
	exifNamespace      = "http://ns.adobe.com/exif/1.0/"
	exifEXNamespace    = "http://cipa.jp/exif/1.0/"
	photoshopNamespace = "http://ns.adobe.com/photoshop/1.0/"
)

type xmpFrame struct {
	kind        string
	text        strings.Builder
	description bool
}

type xmpLocation struct {
	latitude  string
	longitude string
}

func parseXMP(data []byte, source string, collector *collector) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	stack := make([]xmpFrame, 0, 8)
	locations := make([]xmpLocation, 0, 2)
	tokenCount := 0

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		tokenCount++
		if tokenCount > maximumXMPTokens {
			return fmt.Errorf("XMP exceeds the %d-token safety limit", maximumXMPTokens)
		}

		switch value := token.(type) {
		case xml.StartElement:
			if len(stack) >= maximumXMPDepth {
				return fmt.Errorf("XMP exceeds the %d-level nesting safety limit", maximumXMPDepth)
			}
			isDescription := value.Name.Space == rdfNamespace && value.Name.Local == "Description"
			if isDescription {
				locations = append(locations, xmpLocation{})
			}
			stack = append(stack, xmpFrame{kind: xmpPropertyKind(value.Name), description: isDescription})
			for _, attribute := range value.Attr {
				text := safeText(attribute.Value)
				if text == "" {
					continue
				}
				processXMPValue(xmpPropertyKind(attribute.Name), text, source, collector, currentXMPLocation(locations))
			}
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].text.Write([]byte(value))
			}
		case xml.EndElement:
			if len(stack) == 0 {
				continue
			}
			frame := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			text := safeText(frame.text.String())
			if text != "" {
				kind := frame.kind
				if kind == "li" {
					kind = nearestXMPMeaning(stack)
				}
				processXMPValue(kind, text, source, collector, currentXMPLocation(locations))
			}
			if frame.description {
				location := locations[len(locations)-1]
				locations = locations[:len(locations)-1]
				flushXMPLocation(location, source, collector)
			}
		}
	}

	return nil
}

func currentXMPLocation(locations []xmpLocation) *xmpLocation {
	if len(locations) == 0 {
		return nil
	}
	return &locations[len(locations)-1]
}

func flushXMPLocation(location xmpLocation, source string, collector *collector) {
	if location.latitude == "" && location.longitude == "" {
		return
	}
	if location.latitude == "" {
		collector.warn("Ignored incomplete XMP GPS coordinates from %s because latitude is missing.", source)
		return
	}
	if location.longitude == "" {
		collector.warn("Ignored incomplete XMP GPS coordinates from %s because longitude is missing.", source)
		return
	}
	lat, err := parseXMPCoordinate(location.latitude, true)
	if err != nil {
		collector.warn("Could not normalize XMP GPS latitude: %v", err)
		return
	}
	lon, err := parseXMPCoordinate(location.longitude, false)
	if err != nil {
		collector.warn("Could not normalize XMP GPS longitude: %v", err)
		return
	}
	collector.addLocation(lat, lon, source)
}

func processXMPValue(kind, text, source string, collector *collector, location *xmpLocation) {
	switch kind {
	case "capture-time":
		collector.addCaptureTime(text, source)
	case "make":
		collector.addValue(&collector.metadata.DeviceMake, text, source)
	case "model":
		collector.addValue(&collector.metadata.DeviceModel, text, source)
	case "software":
		collector.addValue(&collector.metadata.Software, text, source)
	case "author":
		collector.addValue(&collector.metadata.Authors, text, source)
	case "copyright":
		collector.addValue(&collector.metadata.Copyright, text, source)
	case "comment":
		collector.addValue(&collector.metadata.Comments, text, source)
	case "gps-latitude":
		if location == nil {
			collector.warn("Ignored XMP GPS latitude from %s outside an RDF description.", source)
			return
		}
		location.latitude = text
	case "gps-longitude":
		if location == nil {
			collector.warn("Ignored XMP GPS longitude from %s outside an RDF description.", source)
			return
		}
		location.longitude = text
	}
}

func xmpPropertyKind(name xml.Name) string {
	local := strings.ToLower(name.Local)
	switch name.Space {
	case xmpNamespace:
		switch local {
		case "createdate":
			return "capture-time"
		case "creatortool":
			return "software"
		}
	case photoshopNamespace:
		if local == "datecreated" {
			return "capture-time"
		}
	case tiffNamespace:
		switch local {
		case "make":
			return "make"
		case "model":
			return "model"
		case "software":
			return "software"
		}
	case exifNamespace, exifEXNamespace:
		switch local {
		case "datetimeoriginal":
			return "capture-time"
		case "gpslatitude":
			return "gps-latitude"
		case "gpslongitude":
			return "gps-longitude"
		}
	case dcNamespace:
		switch local {
		case "creator":
			return "author"
		case "rights":
			return "copyright"
		case "description":
			return "comment"
		}
	case rdfNamespace:
		if local == "li" {
			return "li"
		}
	}
	return ""
}

func nearestXMPMeaning(stack []xmpFrame) string {
	for index := len(stack) - 1; index >= 0; index-- {
		switch stack[index].kind {
		case "author", "copyright", "comment":
			return stack[index].kind
		}
	}
	return ""
}

func parseXMPCoordinate(value string, latitude bool) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty coordinate")
	}

	direction := byte(0)
	last := value[len(value)-1]
	if strings.ContainsRune("NnSsEeWw", rune(last)) {
		direction = byte(strings.ToUpper(string(last))[0])
		value = strings.TrimSpace(value[:len(value)-1])
	}
	if latitude && direction != 0 && direction != 'N' && direction != 'S' {
		return 0, fmt.Errorf("latitude has invalid direction %q", direction)
	}
	if !latitude && direction != 0 && direction != 'E' && direction != 'W' {
		return 0, fmt.Errorf("longitude has invalid direction %q", direction)
	}

	replacer := strings.NewReplacer(",", " ", "°", " ", "'", " ", "\"", " ")
	parts := strings.Fields(replacer.Replace(value))
	if len(parts) == 0 || len(parts) > 3 {
		return 0, fmt.Errorf("unsupported coordinate %q", safeText(value))
	}

	numbers := make([]float64, len(parts))
	for index, part := range parts {
		parsed, err := strconv.ParseFloat(part, 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return 0, fmt.Errorf("invalid coordinate %q", safeText(value))
		}
		numbers[index] = parsed
	}

	limit := 180.0
	if latitude {
		limit = 90
	}
	if len(numbers) == 1 {
		coordinate := numbers[0]
		if direction != 0 {
			if coordinate < 0 {
				return 0, fmt.Errorf("signed coordinate cannot also use a direction")
			}
			if direction == 'S' || direction == 'W' {
				coordinate = -coordinate
			}
		}
		if coordinate < -limit || coordinate > limit {
			return 0, fmt.Errorf("coordinate is out of range")
		}
		return coordinate, nil
	}

	coordinate, err := coordinateFromDMS(numbers, limit)
	if err != nil {
		return 0, err
	}
	if direction == 'S' || direction == 'W' {
		coordinate = -coordinate
	}
	return coordinate, nil
}

func coordinateFromDMS(parts []float64, limit float64) (float64, error) {
	if len(parts) < 2 || len(parts) > 3 {
		return 0, fmt.Errorf("expected degrees and minutes with optional seconds")
	}
	for _, part := range parts {
		if math.IsNaN(part) || math.IsInf(part, 0) || part < 0 {
			return 0, fmt.Errorf("coordinate components must be finite and non-negative")
		}
	}
	if parts[1] >= 60 || len(parts) == 3 && parts[2] >= 60 {
		return 0, fmt.Errorf("coordinate minutes and seconds must be less than 60")
	}
	if parts[0] > limit || parts[0] == limit && (parts[1] != 0 || len(parts) == 3 && parts[2] != 0) {
		return 0, fmt.Errorf("coordinate is out of range")
	}
	coordinate := parts[0] + parts[1]/60
	if len(parts) == 3 {
		coordinate += parts[2] / 3600
	}
	return coordinate, nil
}
