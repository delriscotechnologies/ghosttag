package inspect

import (
	"testing"

	"github.com/delriscotechnologies/ghosttag/internal/model"
)

func TestXMPNestedTextDoesNotPanic(t *testing.T) {
	xmp := `<rdf:Description xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:description>prefix<a><b><c><d><e><f><g><h><i>value</i></h></g></f></e></d></c></b></a></dc:description></rdf:Description>`
	var metadata model.Metadata
	var warnings []string
	if err := parseXMP([]byte(xmp), "XMP", newCollector(&metadata, &warnings)); err != nil {
		t.Fatal(err)
	}
	if len(metadata.Comments) != 1 || metadata.Comments[0].Value != "prefix" {
		t.Fatalf("unexpected comments: %+v", metadata.Comments)
	}
}
