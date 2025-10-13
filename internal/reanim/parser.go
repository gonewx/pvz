package reanim

import (
	"encoding/xml"
	"fmt"
	"os"
)

// ParseReanimFile parses a Reanim XML file and returns the animation data.
// Reanim files are XML files without a root element, so this function wraps
// the content with a <reanim> root element before parsing.
//
// Parameters:
//   - path: Path to the Reanim file, e.g., "assets/reanim/PeaShooter.reanim"
//
// Returns:
//   - *ReanimXML: The parsed animation data
//   - error: Parsing error, or nil if successful
//
// Example:
//
//	reanim, err := ParseReanimFile("assets/reanim/PeaShooter.reanim")
//	if err != nil {
//	    log.Fatalf("Failed to parse reanim: %v", err)
//	}
//	fmt.Printf("Animation FPS: %d\n", reanim.FPS)
func ParseReanimFile(path string) (*ReanimXML, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read reanim file '%s': %w", path, err)
	}

	// Wrap the XML content with a root element
	// Original PVZ reanim files don't have a root element, so we add one
	wrappedXML := "<reanim>" + string(data) + "</reanim>"

	// Parse the XML
	var reanim ReanimXML
	if err := xml.Unmarshal([]byte(wrappedXML), &reanim); err != nil {
		return nil, fmt.Errorf("failed to parse XML from '%s': %w", path, err)
	}

	return &reanim, nil
}
