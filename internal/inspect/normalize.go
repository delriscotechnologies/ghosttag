package inspect

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/delriscotechnologies/ghosttag/internal/model"
)

const (
	maximumDisplayRunes   = 500
	maximumMetadataValues = 64
	maximumLocations      = 64
	maximumWarnings       = 100
)

type collector struct {
	metadata          *model.Metadata
	warnings          *[]string
	valueLimitNoted   map[string]bool
	warningLimitNoted bool
}

func newCollector(metadata *model.Metadata, warnings *[]string) *collector {
	return &collector{
		metadata:        metadata,
		warnings:        warnings,
		valueLimitNoted: make(map[string]bool),
	}
}

func (c *collector) addContainer(container string) {
	c.metadata.Containers = appendUnique(c.metadata.Containers, container)
}

func (c *collector) warn(format string, args ...any) {
	if c.warningLimitNoted {
		return
	}
	if len(*c.warnings) >= maximumWarnings-1 {
		*c.warnings = appendUnique(*c.warnings, "Additional parser warnings were omitted after the safety limit was reached.")
		c.warningLimitNoted = true
		return
	}
	warning := safeText(fmt.Sprintf(format, args...))
	if warning != "" {
		*c.warnings = appendUnique(*c.warnings, warning)
	}
}

func (c *collector) noteValueLimit(field string, limit int) {
	if c.valueLimitNoted[field] {
		return
	}
	c.valueLimitNoted[field] = true
	c.warn("Additional %s values were omitted after the %d-value safety limit was reached.", field, limit)
}

func (c *collector) addValue(values *[]model.SourcedValue, value, source string) {
	value = safeText(value)
	if value == "" {
		return
	}
	source = safeText(source)
	for index := range *values {
		if strings.EqualFold((*values)[index].Value, value) {
			(*values)[index].Source = mergeSources((*values)[index].Source, source)
			return
		}
	}
	field := c.valueField(values)
	if len(*values) >= maximumMetadataValues {
		c.noteValueLimit(field, maximumMetadataValues)
		return
	}
	*values = append(*values, model.SourcedValue{Value: value, Source: source})
}

func (c *collector) valueField(values *[]model.SourcedValue) string {
	switch values {
	case &c.metadata.CaptureTime:
		return "capture-time"
	case &c.metadata.DeviceMake:
		return "device-make"
	case &c.metadata.DeviceModel:
		return "device-model"
	case &c.metadata.Software:
		return "software"
	case &c.metadata.Authors:
		return "author"
	case &c.metadata.Copyright:
		return "copyright"
	case &c.metadata.Comments:
		return "comment"
	case &c.metadata.Orientation:
		return "orientation"
	default:
		return "metadata"
	}
}

func (c *collector) addCaptureTime(value, source string) {
	normalized, ok := normalizeTime(value)
	if !ok {
		c.warn("Ignored an invalid capture-time value from %s.", source)
		return
	}
	c.addValue(&c.metadata.CaptureTime, normalized, source)
}

func (c *collector) addLocation(latitude, longitude float64, source string) {
	if math.IsNaN(latitude) || math.IsNaN(longitude) || math.IsInf(latitude, 0) || math.IsInf(longitude, 0) ||
		latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		c.warn("Ignored an invalid or out-of-range GPS coordinate from %s.", source)
		return
	}
	source = safeText(source)
	for index := range c.metadata.Locations {
		existing := &c.metadata.Locations[index]
		if existing.Latitude == latitude && existing.Longitude == longitude {
			existing.Source = mergeSources(existing.Source, source)
			return
		}
	}
	if len(c.metadata.Locations) >= maximumLocations {
		c.noteValueLimit("location", maximumLocations)
		return
	}
	c.metadata.Locations = append(c.metadata.Locations, model.Location{
		Latitude: latitude, Longitude: longitude, Source: source,
	})
}

func mergeSources(existing, source string) string {
	if source == "" {
		return existing
	}
	if existing == "" {
		return source
	}
	for _, item := range strings.Split(existing, " | ") {
		if item == source {
			return existing
		}
	}
	return safeText(existing + " | " + source)
}

func safeText(value string) string {
	var builder strings.Builder
	for _, character := range value {
		if unicode.IsControl(character) || unicode.In(character, unicode.Cf) {
			builder.WriteByte(' ')
			continue
		}
		builder.WriteRune(character)
	}
	value = strings.Join(strings.Fields(builder.String()), " ")
	runes := []rune(value)
	if len(runes) > maximumDisplayRunes {
		return string(runes[:maximumDisplayRunes]) + "…"
	}
	return value
}

func normalizeTime(value string) (string, bool) {
	value = safeText(value)
	if value == "" {
		return "", false
	}

	zonedLayouts := []string{time.RFC3339Nano, time.RFC3339}
	for _, layout := range zonedLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format(time.RFC3339), true
		}
	}

	unzonedLayouts := []string{
		"2006:01:02 15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006:01:02",
		"2006-01-02",
	}
	for _, layout := range unzonedLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			formatted := parsed.Format("2006-01-02")
			if strings.Contains(layout, "15:04:05") {
				formatted = parsed.Format("2006-01-02T15:04:05")
			}
			return formatted + " (timezone not recorded)", true
		}
	}

	return "", false
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
