package assessment

import (
	"fmt"
	"strings"

	"github.com/delriscotechnologies/ghosttag/internal/model"
)

// Evaluate groups supported metadata into privacy-relevant categories. The
// count controls notification wording only; it is not a risk score.
func Evaluate(metadata model.Metadata) model.Assessment {
	categories := make([]string, 0, 5)

	if len(metadata.Locations) > 0 {
		categories = append(categories, "location")
	}
	if len(metadata.CaptureTime) > 0 {
		categories = append(categories, "capture time")
	}
	if len(metadata.DeviceMake) > 0 || len(metadata.DeviceModel) > 0 {
		categories = append(categories, "device")
	}
	if len(metadata.Authors) > 0 || len(metadata.Copyright) > 0 {
		categories = append(categories, "authorship")
	}
	if len(metadata.Comments) > 0 {
		categories = append(categories, "comments")
	}

	assessment := model.Assessment{Categories: categories}
	if len(categories) >= 3 {
		assessment.Notice = fmt.Sprintf(
			"This file contains %d privacy-relevant metadata categories: %s. In combination, these details can reveal more context than each detail alone. Consider whether they are appropriate for the intended recipient or platform.",
			len(categories), strings.Join(categories, ", "),
		)
	}

	return assessment
}
