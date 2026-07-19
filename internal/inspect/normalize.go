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
	maximumMetadataValues = 256
	maximumWarnings       = 100
)

type collector struct {
	metadata          *model.Metadata
	warnings          *[]string
	valuesAdded       int
	valueLimitNoted   bool
	warningLimitNoted bool
}

func newCollector(metadata *model.Metadata, warnings *[]string) *collector {
	return &collector{metadata: metadata, warnings: warnings}
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

func (c *collector) noteValueLimit() {
	if !c.valueLimitNoted {
		c.valueLimitNoted = true
		c.warn("Additional metadata values were omitted after the %d-value safety limit was reached.", maximumMetadataValues)
	}
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
	if c.valuesAdded >= maximumMetadataValues {
		c.noteValueLimit()
		return
	}
	*values = append(*values, model.SourcedValue{Value: value, Source: safeText(source)})
	c.valuesAdded++
}

func (c *collector) addCaptureTime(value, source string) {
	c.addValue(&c.metadata.CaptureTime, normalizeTime(value), source)
}

func (c *collector) addLocation(latitude, longitude float64, source string) {
	if math.IsNaN(latitude) || math.IsNaN(longitude) || math.IsInf(latitude, 0) || math.IsInf(longitude, 0) ||
		latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		c.warn("Ignored an invalid or out-of-range GPS coordinate from %s.", source)
		return
	}
	for _, existing := range c.metadata.Locations {
		if existing.Latitude == latitude && existing.Longitude == longitude {
			return
		}
	}
	if c.valuesAdded >= maximumMetadataValues {
		c.noteValueLimit()
		return
	}
	c.metadata.Locations = append(c.metadata.Locations, model.Location{
		Latitude: latitude, Longitude: longitude, Source: safeText(source),
	})
	c.valuesAdded++
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
