package assessment

import (
	"strings"
	"testing"

	"github.com/delriscotechnologies/ghosttag/internal/model"
)

func TestEvaluateNotificationPolicy(t *testing.T) {
	tests := []struct {
		name             string
		metadata         model.Metadata
		wantCategories   []string
		wantNotice       bool
		forbiddenWording []string
	}{
		{
			name:           "no relevant metadata",
			metadata:       model.Metadata{Software: []model.SourcedValue{{Value: "Editor", Source: "XMP"}}},
			wantCategories: nil,
		},
		{
			name: "two categories are listed without a notice",
			metadata: model.Metadata{
				DeviceModel: []model.SourcedValue{{Value: "Camera", Source: "EXIF"}},
				Comments:    []model.SourcedValue{{Value: "Draft", Source: "XMP"}},
			},
			wantCategories: []string{"device", "comments"},
		},
		{
			name: "three categories receive a combined-context notice",
			metadata: model.Metadata{
				Locations:   []model.Location{{Latitude: 1, Longitude: 2, Source: "EXIF"}},
				CaptureTime: []model.SourcedValue{{Value: "2026-07-18", Source: "EXIF"}},
				Authors:     []model.SourcedValue{{Value: "A. Person", Source: "XMP"}},
			},
			wantCategories:   []string{"location", "capture time", "authorship"},
			wantNotice:       true,
			forbiddenWording: []string{"high", "medium", "low", "score", "severity"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.metadata)
			if strings.Join(got.Categories, ",") != strings.Join(test.wantCategories, ",") {
				t.Fatalf("categories = %v, want %v", got.Categories, test.wantCategories)
			}
			if (got.Notice != "") != test.wantNotice {
				t.Fatalf("notice = %q, wantNotice %v", got.Notice, test.wantNotice)
			}
			lowerNotice := strings.ToLower(got.Notice)
			for _, forbidden := range test.forbiddenWording {
				if strings.Contains(lowerNotice, forbidden) {
					t.Fatalf("notice contains forbidden wording %q: %s", forbidden, got.Notice)
				}
			}
		})
	}
}
