package particle

import (
	"encoding/xml"
	"fmt"

	"github.com/decker502/pvz/pkg/embedded"
)

// ParseParticleXML parses a particle configuration XML file and returns the parsed configuration.
//
// The XML file may contain multiple top-level <Emitter> elements without a root wrapper.
// This function handles such cases by wrapping the content in a synthetic root element.
//
// Parameters:
//   - path: File path to the particle XML configuration file
//
// Returns:
//   - *ParticleConfig: Parsed configuration containing all emitters
//   - error: Any error encountered during file reading or XML parsing
//
// Example usage:
//
//	config, err := ParseParticleXML("data/particles/Award.xml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Loaded %d emitters\n", len(config.Emitters))
func ParseParticleXML(path string) (*ParticleConfig, error) {
	// 从 embedded FS 读取 XML 文件
	data, err := embedded.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read particle XML file %s: %w", path, err)
	}

	// Wrap the content with a root element since particle XML files
	// contain multiple top-level <Emitter> elements without a root wrapper
	wrappedXML := fmt.Sprintf("<ParticleConfig>%s</ParticleConfig>", string(data))

	// Parse the XML into our data structure
	var config ParticleConfig
	if err := xml.Unmarshal([]byte(wrappedXML), &config); err != nil {
		return nil, fmt.Errorf("failed to parse particle XML %s: %w", path, err)
	}

	// Validate that we parsed at least one emitter
	if len(config.Emitters) == 0 {
		return nil, fmt.Errorf("particle XML %s contains no emitters", path)
	}

	return &config, nil
}
