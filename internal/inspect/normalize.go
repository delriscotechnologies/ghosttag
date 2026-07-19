package inspect

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/delriscotechnologies/ghosttag/internal/model"
)

const maximumDisplayRunes = 500

type collector struct {
	metadata *model.Metadata
	warnings *[]string
}

func newCollector(metadata *model.Metadata, warnings *[]string) *collector {
	return &collector{metadata: metadata, warnings: warnings}
}

func (c *collector) addContainer(container string) {
	c.metadata.Containers = appendUnique(c.metadata.Containers, container)
}

func (c *collector) warn(format string, args ...any) {
	*c.warnings = appendUnique(*c.warnings, fmt.Sprintf(format, args...))
}

func (c *collector) addValue(values *[]model.SourcedValue, value, source string) {
	value = safeText(value)
	if value == "" {
		return
	}
	for _, existing := range *values {
		if strings.EqualFold(existing.Value, value) {
			return
		}
	}
	*values = append(*values, model.SourcedValue{Value: value, Source: source})
}

func (c *collector) addCaptureTime(value, source string) {
	c.addValue(&c.metadata.CaptureTime, normalizeTime(value), source)
}

func (c *collector) addLocation(latitude, longitude float64, source string) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		c.warn("Ignored an out-of-range GPS coordinate from %s.", source)
		return
	}
	for _, existing := range c.metadata.Locations {
		if existing.Latitude == latitude && existing.Longitude == longitude {
			return
		}
	}
	c.metadata.Locations = append(c.metadata.Locations, model.Location{
		Latitude: latitude, Longitude: longitude, Source: source,
	})
}

func safeText(value string) string {
	var builder strings.Builder
	for _, character := range value {
		if unicode.IsControl(character) {
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

func normalizeTime(value string) string {
	value = safeText(value)
	if value == "" {
		return ""
	}

	zonedLayouts := []string{time.RFC3339Nano, time.RFC3339}
	for _, layout := range zonedLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format(time.RFC3339)
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
			return formatted + " (timezone not recorded)"
		}
	}

	return value
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
