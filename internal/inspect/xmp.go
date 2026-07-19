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
)

type xmpFrame struct {
	name string
	text strings.Builder
}

func parseXMP(data []byte, source string, collector *collector) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	stack := make([]xmpFrame, 0, 8)
	latitude, longitude := "", ""
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
			stack = append(stack, xmpFrame{name: strings.ToLower(value.Name.Local)})
			for _, attribute := range value.Attr {
				text := safeText(attribute.Value)
				if text == "" {
					continue
				}
				switch strings.ToLower(attribute.Name.Local) {
				case "createdate", "datetimeoriginal", "datecreated":
					collector.addCaptureTime(text, source)
				case "make":
					collector.addValue(&collector.metadata.DeviceMake, text, source)
				case "model":
					collector.addValue(&collector.metadata.DeviceModel, text, source)
				case "creatortool", "software":
					collector.addValue(&collector.metadata.Software, text, source)
				case "artist", "author", "creator":
					collector.addValue(&collector.metadata.Authors, text, source)
				case "copyright", "rights":
					collector.addValue(&collector.metadata.Copyright, text, source)
				case "comment", "caption", "description":
					collector.addValue(&collector.metadata.Comments, text, source)
				case "gpslatitude":
					latitude = text
				case "gpslongitude":
					longitude = text
				}
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
			if text == "" {
				continue
			}

			switch frame.name {
			case "createdate", "datetimeoriginal", "datecreated":
				collector.addCaptureTime(text, source)
			case "make":
				collector.addValue(&collector.metadata.DeviceMake, text, source)
			case "model":
				collector.addValue(&collector.metadata.DeviceModel, text, source)
			case "creatortool", "software":
				collector.addValue(&collector.metadata.Software, text, source)
			case "artist", "author", "creator":
				collector.addValue(&collector.metadata.Authors, text, source)
			case "copyright", "rights":
				collector.addValue(&collector.metadata.Copyright, text, source)
			case "comment", "caption", "description":
				collector.addValue(&collector.metadata.Comments, text, source)
			case "gpslatitude":
				latitude = text
			case "gpslongitude":
				longitude = text
			case "li":
				switch nearestXMPMeaning(stack) {
				case "creator":
					collector.addValue(&collector.metadata.Authors, text, source)
				case "rights":
					collector.addValue(&collector.metadata.Copyright, text, source)
				case "description":
					collector.addValue(&collector.metadata.Comments, text, source)
				}
			}
		}
	}

	if latitude != "" && longitude != "" {
		lat, err := parseXMPCoordinate(latitude, true)
		if err != nil {
			collector.warn("Could not normalize XMP GPS latitude: %v", err)
			return nil
		}
		lon, err := parseXMPCoordinate(longitude, false)
		if err != nil {
			collector.warn("Could not normalize XMP GPS longitude: %v", err)
			return nil
		}
		collector.addLocation(lat, lon, source)
	}
	return nil
}

func nearestXMPMeaning(stack []xmpFrame) string {
	for index := len(stack) - 1; index >= 0; index-- {
		switch stack[index].name {
		case "creator", "rights", "description":
			return stack[index].name
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
