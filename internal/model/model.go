package model

// SourcedValue is a normalized metadata value and the container it came from.
type SourcedValue struct {
	Value  string
	Source string
}

// Location is a normalized decimal GPS coordinate.
type Location struct {
	Latitude  float64
	Longitude float64
	Source    string
}

// FileInfo contains facts derived from the file itself.
type FileInfo struct {
	Name           string
	Extension      string
	DetectedFormat string
	Size           int64
	SHA256         string
	Width          int
	Height         int
}

// Metadata contains supported normalized facts grouped by meaning rather than
// by the source tag name.
type Metadata struct {
	Containers  []string
	Locations   []Location
	CaptureTime []SourcedValue
	DeviceMake  []SourcedValue
	DeviceModel []SourcedValue
	Software    []SourcedValue
	Authors     []SourcedValue
	Copyright   []SourcedValue
	Comments    []SourcedValue
	Orientation []SourcedValue
}

// Assessment describes which privacy-relevant categories were present. It is
// deliberately not a score or severity rating.
type Assessment struct {
	Categories []string
	Notice     string
}

// Report is the complete result for one input file.
type Report struct {
	File       FileInfo
	Metadata   Metadata
	Warnings   []string
	Assessment Assessment
}

// HasSupportedFields reports whether any supported metadata value was found.
func (m Metadata) HasSupportedFields() bool {
	return len(m.Locations) > 0 ||
		len(m.CaptureTime) > 0 ||
		len(m.DeviceMake) > 0 ||
		len(m.DeviceModel) > 0 ||
		len(m.Software) > 0 ||
		len(m.Authors) > 0 ||
		len(m.Copyright) > 0 ||
		len(m.Comments) > 0 ||
		len(m.Orientation) > 0
}
