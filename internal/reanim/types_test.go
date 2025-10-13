package reanim

import (
	"encoding/xml"
	"testing"
)

// TestReanimXML_XMLUnmarshaling tests that ReanimXML structure correctly
// unmarshals from XML with proper tag mappings.
func TestReanimXML_XMLUnmarshaling(t *testing.T) {
	xmlData := `<reanim>
		<fps>12</fps>
		<track>
			<name>anim_idle</name>
			<t><f>0</f></t>
		</track>
		<track>
			<name>head</name>
			<t><x>10.5</x><y>20.3</y></t>
		</track>
	</reanim>`

	var reanim ReanimXML
	err := xml.Unmarshal([]byte(xmlData), &reanim)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	// Verify FPS
	if reanim.FPS != 12 {
		t.Errorf("Expected FPS=12, got %d", reanim.FPS)
	}

	// Verify Tracks length
	if len(reanim.Tracks) != 2 {
		t.Errorf("Expected 2 tracks, got %d", len(reanim.Tracks))
	}

	// Verify first track
	if reanim.Tracks[0].Name != "anim_idle" {
		t.Errorf("Expected first track name='anim_idle', got '%s'", reanim.Tracks[0].Name)
	}
	if len(reanim.Tracks[0].Frames) != 1 {
		t.Errorf("Expected 1 frame in first track, got %d", len(reanim.Tracks[0].Frames))
	}

	// Verify second track
	if reanim.Tracks[1].Name != "head" {
		t.Errorf("Expected second track name='head', got '%s'", reanim.Tracks[1].Name)
	}
}

// TestFrame_PointerFields tests that Frame fields correctly handle null values
// using pointer types.
func TestFrame_PointerFields(t *testing.T) {
	tests := []struct {
		name        string
		xmlData     string
		expectNil   []string // Fields that should be nil
		expectValue map[string]interface{}
	}{
		{
			name:      "Empty frame",
			xmlData:   `<t></t>`,
			expectNil: []string{"FrameNum", "X", "Y", "ScaleX", "ScaleY", "SkewX", "SkewY"},
		},
		{
			name:      "Frame with FrameNum only",
			xmlData:   `<t><f>-1</f></t>`,
			expectNil: []string{"X", "Y", "ScaleX", "ScaleY", "SkewX", "SkewY"},
			expectValue: map[string]interface{}{
				"FrameNum": -1,
			},
		},
		{
			name:      "Frame with position",
			xmlData:   `<t><x>50.5</x><y>100.7</y></t>`,
			expectNil: []string{"FrameNum", "ScaleX", "ScaleY", "SkewX", "SkewY"},
			expectValue: map[string]interface{}{
				"X": 50.5,
				"Y": 100.7,
			},
		},
		{
			name:      "Frame with scale and skew",
			xmlData:   `<t><sx>2.0</sx><sy>1.5</sy><kx>0.1</kx><ky>0.2</ky></t>`,
			expectNil: []string{"FrameNum", "X", "Y"},
			expectValue: map[string]interface{}{
				"ScaleX": 2.0,
				"ScaleY": 1.5,
				"SkewX":  0.1,
				"SkewY":  0.2,
			},
		},
		{
			name:      "Frame with image reference",
			xmlData:   `<t><i>IMAGE_REANIM_PEASHOOTER_HEAD</i></t>`,
			expectNil: []string{"FrameNum", "X", "Y", "ScaleX", "ScaleY", "SkewX", "SkewY"},
			expectValue: map[string]interface{}{
				"ImagePath": "IMAGE_REANIM_PEASHOOTER_HEAD",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var frame Frame
			err := xml.Unmarshal([]byte(tt.xmlData), &frame)
			if err != nil {
				t.Fatalf("Failed to unmarshal XML: %v", err)
			}

			// Check nil fields
			for _, field := range tt.expectNil {
				switch field {
				case "FrameNum":
					if frame.FrameNum != nil {
						t.Errorf("Expected %s to be nil, got %v", field, *frame.FrameNum)
					}
				case "X":
					if frame.X != nil {
						t.Errorf("Expected %s to be nil, got %v", field, *frame.X)
					}
				case "Y":
					if frame.Y != nil {
						t.Errorf("Expected %s to be nil, got %v", field, *frame.Y)
					}
				case "ScaleX":
					if frame.ScaleX != nil {
						t.Errorf("Expected %s to be nil, got %v", field, *frame.ScaleX)
					}
				case "ScaleY":
					if frame.ScaleY != nil {
						t.Errorf("Expected %s to be nil, got %v", field, *frame.ScaleY)
					}
				case "SkewX":
					if frame.SkewX != nil {
						t.Errorf("Expected %s to be nil, got %v", field, *frame.SkewX)
					}
				case "SkewY":
					if frame.SkewY != nil {
						t.Errorf("Expected %s to be nil, got %v", field, *frame.SkewY)
					}
				}
			}

			// Check expected values
			for field, expectedValue := range tt.expectValue {
				switch field {
				case "FrameNum":
					if frame.FrameNum == nil {
						t.Errorf("Expected %s to have value %v, got nil", field, expectedValue)
					} else if *frame.FrameNum != expectedValue.(int) {
						t.Errorf("Expected %s=%v, got %v", field, expectedValue, *frame.FrameNum)
					}
				case "X":
					if frame.X == nil {
						t.Errorf("Expected %s to have value %v, got nil", field, expectedValue)
					} else if *frame.X != expectedValue.(float64) {
						t.Errorf("Expected %s=%v, got %v", field, expectedValue, *frame.X)
					}
				case "Y":
					if frame.Y == nil {
						t.Errorf("Expected %s to have value %v, got nil", field, expectedValue)
					} else if *frame.Y != expectedValue.(float64) {
						t.Errorf("Expected %s=%v, got %v", field, expectedValue, *frame.Y)
					}
				case "ScaleX":
					if frame.ScaleX == nil {
						t.Errorf("Expected %s to have value %v, got nil", field, expectedValue)
					} else if *frame.ScaleX != expectedValue.(float64) {
						t.Errorf("Expected %s=%v, got %v", field, expectedValue, *frame.ScaleX)
					}
				case "ScaleY":
					if frame.ScaleY == nil {
						t.Errorf("Expected %s to have value %v, got nil", field, expectedValue)
					} else if *frame.ScaleY != expectedValue.(float64) {
						t.Errorf("Expected %s=%v, got %v", field, expectedValue, *frame.ScaleY)
					}
				case "SkewX":
					if frame.SkewX == nil {
						t.Errorf("Expected %s to have value %v, got nil", field, expectedValue)
					} else if *frame.SkewX != expectedValue.(float64) {
						t.Errorf("Expected %s=%v, got %v", field, expectedValue, *frame.SkewX)
					}
				case "SkewY":
					if frame.SkewY == nil {
						t.Errorf("Expected %s to have value %v, got nil", field, expectedValue)
					} else if *frame.SkewY != expectedValue.(float64) {
						t.Errorf("Expected %s=%v, got %v", field, expectedValue, *frame.SkewY)
					}
				case "ImagePath":
					if frame.ImagePath != expectedValue.(string) {
						t.Errorf("Expected %s=%v, got %v", field, expectedValue, frame.ImagePath)
					}
				}
			}
		})
	}
}

// TestFrame_StructAccess tests that Frame struct fields can be accessed normally.
func TestFrame_StructAccess(t *testing.T) {
	// Create a frame with all fields set
	frameNum := 0
	x := 10.5
	y := 20.3
	scaleX := 1.5
	scaleY := 2.0
	skewX := 0.1
	skewY := 0.2

	frame := Frame{
		FrameNum:  &frameNum,
		X:         &x,
		Y:         &y,
		ScaleX:    &scaleX,
		ScaleY:    &scaleY,
		SkewX:     &skewX,
		SkewY:     &skewY,
		ImagePath: "IMAGE_REANIM_TEST",
	}

	// Access and verify all fields
	if frame.FrameNum == nil || *frame.FrameNum != 0 {
		t.Errorf("Expected FrameNum=0, got %v", frame.FrameNum)
	}
	if frame.X == nil || *frame.X != 10.5 {
		t.Errorf("Expected X=10.5, got %v", frame.X)
	}
	if frame.Y == nil || *frame.Y != 20.3 {
		t.Errorf("Expected Y=20.3, got %v", frame.Y)
	}
	if frame.ScaleX == nil || *frame.ScaleX != 1.5 {
		t.Errorf("Expected ScaleX=1.5, got %v", frame.ScaleX)
	}
	if frame.ScaleY == nil || *frame.ScaleY != 2.0 {
		t.Errorf("Expected ScaleY=2.0, got %v", frame.ScaleY)
	}
	if frame.SkewX == nil || *frame.SkewX != 0.1 {
		t.Errorf("Expected SkewX=0.1, got %v", frame.SkewX)
	}
	if frame.SkewY == nil || *frame.SkewY != 0.2 {
		t.Errorf("Expected SkewY=0.2, got %v", frame.SkewY)
	}
	if frame.ImagePath != "IMAGE_REANIM_TEST" {
		t.Errorf("Expected ImagePath='IMAGE_REANIM_TEST', got '%s'", frame.ImagePath)
	}
}
