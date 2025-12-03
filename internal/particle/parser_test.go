package particle

import (
	"os"
	"testing"
)

// TestParseParticleXML_Award tests parsing the Award.xml file
func TestParseParticleXML_Award(t *testing.T) {
	config, err := ParseParticleXML("../../data/particles/Award.xml")
	if err != nil {
		t.Fatalf("Failed to parse Award.xml: %v", err)
	}

	if config == nil {
		t.Fatal("Parsed config is nil")
	}

	// Award.xml should contain 13 emitters
	expectedEmitters := 13
	if len(config.Emitters) != expectedEmitters {
		t.Errorf("Expected %d emitters, got %d", expectedEmitters, len(config.Emitters))
	}

	// Check first emitter name
	if len(config.Emitters) > 0 {
		firstEmitter := config.Emitters[0]
		if firstEmitter.Name != "AwardRay8" {
			t.Errorf("Expected first emitter name 'AwardRay8', got '%s'", firstEmitter.Name)
		}

		// Verify some key properties are parsed
		if firstEmitter.Image != "IMAGE_AWARDRAYS2" {
			t.Errorf("Expected image 'IMAGE_AWARDRAYS2', got '%s'", firstEmitter.Image)
		}

		if firstEmitter.Additive != "1" {
			t.Errorf("Expected Additive '1', got '%s'", firstEmitter.Additive)
		}
	}
}

// TestParseParticleXML_BossExplosion tests parsing the BossExplosion.xml file
func TestParseParticleXML_BossExplosion(t *testing.T) {
	config, err := ParseParticleXML("../../data/particles/BossExplosion.xml")
	if err != nil {
		t.Fatalf("Failed to parse BossExplosion.xml: %v", err)
	}

	if config == nil {
		t.Fatal("Parsed config is nil")
	}

	// BossExplosion.xml should contain 5 emitters
	expectedEmitters := 5
	if len(config.Emitters) != expectedEmitters {
		t.Errorf("Expected %d emitters, got %d", expectedEmitters, len(config.Emitters))
	}

	// Check for Field in first emitter (Parts)
	if len(config.Emitters) > 0 {
		firstEmitter := config.Emitters[0]
		if firstEmitter.Name != "Parts" {
			t.Errorf("Expected first emitter name 'Parts', got '%s'", firstEmitter.Name)
		}

		// Verify Field is parsed
		if len(firstEmitter.Fields) != 1 {
			t.Errorf("Expected 1 field, got %d", len(firstEmitter.Fields))
		} else {
			field := firstEmitter.Fields[0]
			if field.FieldType != "Acceleration" {
				t.Errorf("Expected FieldType 'Acceleration', got '%s'", field.FieldType)
			}
			if field.Y != "13" {
				t.Errorf("Expected Y '13', got '%s'", field.Y)
			}
		}
	}
}

// TestParseParticleXML_NonExistent tests error handling for missing files
func TestParseParticleXML_NonExistent(t *testing.T) {
	_, err := ParseParticleXML("nonexistent.xml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestParseParticleXML_CabbageSplat tests parsing CabbageSplat.xml (multiple Fields)
func TestParseParticleXML_CabbageSplat(t *testing.T) {
	config, err := ParseParticleXML("../../data/particles/CabbageSplat.xml")
	if err != nil {
		t.Fatalf("Failed to parse CabbageSplat.xml: %v", err)
	}

	// Should contain 2 emitters
	expectedEmitters := 2
	if len(config.Emitters) != expectedEmitters {
		t.Errorf("Expected %d emitters, got %d", expectedEmitters, len(config.Emitters))
	}

	// Check second emitter (PeaSplatBits) has multiple Fields
	if len(config.Emitters) > 1 {
		secondEmitter := config.Emitters[1]
		if secondEmitter.Name != "PeaSplatBits" {
			t.Errorf("Expected second emitter name 'PeaSplatBits', got '%s'", secondEmitter.Name)
		}

		// Should have 2 fields (Friction and Acceleration)
		if len(secondEmitter.Fields) != 2 {
			t.Errorf("Expected 2 fields, got %d", len(secondEmitter.Fields))
		} else {
			// Check first field (Friction)
			if secondEmitter.Fields[0].FieldType != "Friction" {
				t.Errorf("Expected first FieldType 'Friction', got '%s'", secondEmitter.Fields[0].FieldType)
			}
			// Check second field (Acceleration)
			if secondEmitter.Fields[1].FieldType != "Acceleration" {
				t.Errorf("Expected second FieldType 'Acceleration', got '%s'", secondEmitter.Fields[1].FieldType)
			}
		}
	}
}

// TestParseParticleXML_BlastMark tests parsing BlastMark.xml (simple single emitter)
func TestParseParticleXML_BlastMark(t *testing.T) {
	config, err := ParseParticleXML("../../data/particles/BlastMark.xml")
	if err != nil {
		t.Fatalf("Failed to parse BlastMark.xml: %v", err)
	}

	// Should contain 1 emitter
	expectedEmitters := 1
	if len(config.Emitters) != expectedEmitters {
		t.Errorf("Expected %d emitters, got %d", expectedEmitters, len(config.Emitters))
	}

	// Check emitter properties
	if len(config.Emitters) > 0 {
		emitter := config.Emitters[0]

		// BlastMark emitter doesn't have a Name tag
		if emitter.Name != "" {
			t.Errorf("Expected empty name, got '%s'", emitter.Name)
		}

		if emitter.Image != "IMAGE_BLASTMARK" {
			t.Errorf("Expected image 'IMAGE_BLASTMARK', got '%s'", emitter.Image)
		}

		if emitter.SystemDuration != "100" {
			t.Errorf("Expected SystemDuration '100', got '%s'", emitter.SystemDuration)
		}
	}
}

// TestParseParticleXML_CoinPickupArrow tests parsing CoinPickupArrow.xml (UI effect with Position field)
func TestParseParticleXML_CoinPickupArrow(t *testing.T) {
	config, err := ParseParticleXML("../../data/particles/CoinPickupArrow.xml")
	if err != nil {
		t.Fatalf("Failed to parse CoinPickupArrow.xml: %v", err)
	}

	// Should contain 1 emitter
	expectedEmitters := 1
	if len(config.Emitters) != expectedEmitters {
		t.Errorf("Expected %d emitters, got %d", expectedEmitters, len(config.Emitters))
	}

	// Check for Position field with complex interpolation
	if len(config.Emitters) > 0 {
		emitter := config.Emitters[0]

		if len(emitter.Fields) != 1 {
			t.Errorf("Expected 1 field, got %d", len(emitter.Fields))
		} else {
			field := emitter.Fields[0]
			if field.FieldType != "Position" {
				t.Errorf("Expected FieldType 'Position', got '%s'", field.FieldType)
			}

			// Verify complex Y value with Linear interpolation is preserved as string
			expectedY := "0 Linear 10,50 Linear 0"
			if field.Y != expectedY {
				t.Errorf("Expected Y '%s', got '%s'", expectedY, field.Y)
			}
		}

		// Verify EmitterOffsetX is parsed
		if emitter.EmitterOffsetX != "-11" {
			t.Errorf("Expected EmitterOffsetX '-11', got '%s'", emitter.EmitterOffsetX)
		}
	}
}

// TestParseParticleXML_InvalidXML tests error handling for malformed XML
func TestParseParticleXML_InvalidXML(t *testing.T) {
	// Create a temporary invalid XML file
	tmpFile := "/tmp/test_invalid_particle.xml"
	invalidXML := []byte("<Emitter><Name>Test</Name>") // Missing closing tag

	err := os.WriteFile(tmpFile, invalidXML, 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	_, err = ParseParticleXML(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid XML, got nil")
	}
}
