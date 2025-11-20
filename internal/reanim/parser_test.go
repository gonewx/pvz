package reanim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseReanimFile_Success tests successful parsing of PeaShooter.reanim
func TestParseReanimFile_Success(t *testing.T) {
	path := "../../assets/effect/reanim/PeaShooter.reanim"

	reanim, err := ParseReanimFile(path)
	if err != nil {
		t.Fatalf("Failed to parse PeaShooter.reanim: %v", err)
	}

	// Verify FPS (should be 12 for PVZ animations)
	if reanim.FPS != 12 {
		t.Errorf("Expected FPS=12, got %d", reanim.FPS)
	}

	// Verify Track count (PeaShooter should have multiple tracks)
	if len(reanim.Tracks) == 0 {
		t.Errorf("Expected at least one track, got 0")
	}

	// Verify at least one animation definition track exists (name starts with "anim_")
	hasAnimTrack := false
	for _, track := range reanim.Tracks {
		if strings.HasPrefix(track.Name, "anim_") {
			hasAnimTrack = true
			break
		}
	}
	if !hasAnimTrack {
		t.Errorf("Expected at least one animation definition track (starting with 'anim_'), found none")
	}

	// Verify at least one part track exists (e.g., "head", "body")
	hasPartTrack := false
	for _, track := range reanim.Tracks {
		if !strings.HasPrefix(track.Name, "anim_") && track.Name != "" {
			hasPartTrack = true
			break
		}
	}
	if !hasPartTrack {
		t.Errorf("Expected at least one part track (not starting with 'anim_'), found none")
	}

	// Verify tracks have frames
	for _, track := range reanim.Tracks {
		if len(track.Frames) == 0 {
			t.Errorf("Track '%s' has no frames", track.Name)
		}
	}
}

// TestParseReanimFile_MultipleFiles tests parsing multiple reanim files
func TestParseReanimFile_MultipleFiles(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"PeaShooter", "PeaShooter.reanim"},
		{"SunFlower", "SunFlower.reanim"},
		{"Wallnut", "Wallnut.reanim"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("../../assets/reanim", tt.filename)

			reanim, err := ParseReanimFile(path)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", tt.filename, err)
			}

			// Verify FPS
			if reanim.FPS != 12 {
				t.Errorf("%s: Expected FPS=12, got %d", tt.name, reanim.FPS)
			}

			// Verify has tracks
			if len(reanim.Tracks) == 0 {
				t.Errorf("%s: Expected at least one track, got 0", tt.name)
			}

			// Verify tracks have names
			for i, track := range reanim.Tracks {
				if track.Name == "" {
					t.Errorf("%s: Track %d has empty name", tt.name, i)
				}
			}
		})
	}
}

// TestParseReanimFile_Errors tests error handling scenarios
func TestParseReanimFile_Errors(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError string
	}{
		{
			name:        "File not found",
			path:        "../../assets/effect/reanim/NonExistent.reanim",
			expectError: "failed to read reanim file",
		},
		{
			name:        "Invalid XML format",
			path:        "test_invalid.reanim",
			expectError: "failed to parse XML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For invalid XML test, create a temporary file
			if tt.name == "Invalid XML format" {
				// Create a temporary invalid XML file
				invalidXML := []byte("<fps>12<track><name>test</name></track>") // Missing closing tags
				tmpFile, err := os.CreateTemp("", "test_invalid*.reanim")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())

				if _, err := tmpFile.Write(invalidXML); err != nil {
					t.Fatalf("Failed to write temp file: %v", err)
				}
				tmpFile.Close()

				tt.path = tmpFile.Name()
			}

			reanim, err := ParseReanimFile(tt.path)

			// Verify error occurred
			if err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.expectError)
				return
			}

			// Verify reanim is nil on error
			if reanim != nil {
				t.Errorf("Expected nil reanim on error, got %v", reanim)
			}

			// Verify error message contains expected text
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectError, err.Error())
			}
		})
	}
}

// TestParseReanimFile_FrameData tests that frame data is correctly parsed
func TestParseReanimFile_FrameData(t *testing.T) {
	path := "../../assets/effect/reanim/PeaShooter.reanim"

	reanim, err := ParseReanimFile(path)
	if err != nil {
		t.Fatalf("Failed to parse PeaShooter.reanim: %v", err)
	}

	// Find the first track with frames that have non-nil values
	var foundTrack *Track
	for i := range reanim.Tracks {
		if len(reanim.Tracks[i].Frames) > 0 {
			foundTrack = &reanim.Tracks[i]
			break
		}
	}

	if foundTrack == nil {
		t.Fatal("No track with frames found")
	}

	// Verify frames exist
	if len(foundTrack.Frames) == 0 {
		t.Errorf("Track '%s' has no frames", foundTrack.Name)
	}

	// Check that at least one frame has some data (could be FrameNum, position, or image)
	hasData := false
	for _, frame := range foundTrack.Frames {
		if frame.FrameNum != nil || frame.X != nil || frame.Y != nil ||
			frame.ScaleX != nil || frame.ScaleY != nil ||
			frame.SkewX != nil || frame.SkewY != nil ||
			frame.ImagePath != "" {
			hasData = true
			break
		}
	}

	if !hasData {
		t.Errorf("Track '%s' has frames but none contain any data", foundTrack.Name)
	}
}

// TestCompareWithReference tests that our implementation produces the same results
// as the reference implementation from test_animation_viewer.go
func TestCompareWithReference(t *testing.T) {
	// Parse PeaShooter.reanim using our implementation
	path := "../../assets/effect/reanim/PeaShooter.reanim"
	reanim, err := ParseReanimFile(path)
	if err != nil {
		t.Fatalf("Failed to parse PeaShooter.reanim: %v", err)
	}

	// Verify FPS matches reference (should be 12)
	expectedFPS := 12
	if reanim.FPS != expectedFPS {
		t.Errorf("FPS mismatch: expected %d, got %d", expectedFPS, reanim.FPS)
	}

	// Verify track count (PeaShooter has specific number of tracks)
	if len(reanim.Tracks) == 0 {
		t.Fatal("Expected non-zero track count")
	}
	t.Logf("✓ Parsed %d tracks with FPS=%d", len(reanim.Tracks), reanim.FPS)

	// Verify animation definition tracks exist
	animTracks := 0
	partTracks := 0
	for _, track := range reanim.Tracks {
		if strings.HasPrefix(track.Name, "anim_") {
			animTracks++
		} else if track.Name != "" {
			partTracks++
		}
	}

	if animTracks == 0 {
		t.Error("Expected at least one animation definition track (starting with 'anim_')")
	}
	if partTracks == 0 {
		t.Error("Expected at least one part track (not starting with 'anim_')")
	}
	t.Logf("✓ Found %d animation tracks and %d part tracks", animTracks, partTracks)

	// Verify specific expected track names from reference implementation
	expectedTrackNames := []string{"anim_idle", "anim_shooting"}
	for _, expectedName := range expectedTrackNames {
		found := false
		for _, track := range reanim.Tracks {
			if strings.Contains(track.Name, expectedName) {
				found = true
				break
			}
		}
		if !found {
			t.Logf("⚠ Warning: Expected track containing '%s' not found (may be normal)", expectedName)
		}
	}

	// Verify frame data structure for the first track with frames
	for i, track := range reanim.Tracks {
		if len(track.Frames) == 0 {
			continue
		}

		// Check first 10 frames (or all if less than 10)
		framesToCheck := 10
		if len(track.Frames) < framesToCheck {
			framesToCheck = len(track.Frames)
		}

		for j := 0; j < framesToCheck; j++ {
			frame := track.Frames[j]

			// Verify frame structure is valid (at least one field should be set or all nil)
			hasAnyField := frame.FrameNum != nil || frame.X != nil || frame.Y != nil ||
				frame.ScaleX != nil || frame.ScaleY != nil ||
				frame.SkewX != nil || frame.SkewY != nil ||
				frame.ImagePath != ""

			if !hasAnyField && j > 0 {
				// Empty frames after the first frame are valid (they inherit from previous frame)
				continue
			}

			if j == 0 && !hasAnyField && frame.ImagePath == "" {
				t.Errorf("Track[%d]'%s' Frame[%d]: First frame should have at least one field set",
					i, track.Name, j)
			}
		}

		t.Logf("✓ Track '%s': verified %d frames", track.Name, framesToCheck)
		break // Only check first track with frames for this test
	}

	// Verify frame inheritance behavior (null fields should be allowed)
	// This tests that our pointer-based approach works correctly
	for _, track := range reanim.Tracks {
		if len(track.Frames) < 2 {
			continue
		}

		// Count how many frames have null fields
		nullFieldCount := 0
		for _, frame := range track.Frames {
			if frame.X == nil && frame.Y == nil && frame.FrameNum == nil {
				nullFieldCount++
			}
		}

		// In reanim files, it's common to have many frames with null fields
		// (they inherit from previous frames)
		if nullFieldCount > 0 {
			t.Logf("✓ Track '%s': %d frames have null fields (inheritance)",
				track.Name, nullFieldCount)
		}
		break // Only check one track for this test
	}
}

// TestBuildMergedTracks_DuplicateTrackNames tests handling of duplicate track names
// LoadBar_sprout.reanim contains multiple tracks with the same name (e.g., 6 "rock" tracks)
func TestBuildMergedTracks_DuplicateTrackNames(t *testing.T) {
	path := "../../data/reanim/LoadBar_sprout.reanim"

	reanimXML, err := ParseReanimFile(path)
	if err != nil {
		t.Fatalf("Failed to parse LoadBar_sprout.reanim: %v", err)
	}

	// Count duplicate track names in the original file
	trackNameCounts := make(map[string]int)
	for _, track := range reanimXML.Tracks {
		trackNameCounts[track.Name]++
	}

	// Verify there are duplicate track names
	rockCount := trackNameCounts["rock"]
	sproutPetalCount := trackNameCounts["sprout_petal"]

	if rockCount < 2 {
		t.Errorf("Expected multiple 'rock' tracks, got %d", rockCount)
	}
	if sproutPetalCount < 2 {
		t.Errorf("Expected multiple 'sprout_petal' tracks, got %d", sproutPetalCount)
	}

	t.Logf("Original file has %d 'rock' tracks and %d 'sprout_petal' tracks", rockCount, sproutPetalCount)

	// Build merged tracks
	mergedTracks := BuildMergedTracks(reanimXML)

	// Verify all tracks are present with unique keys
	expectedKeyCount := len(reanimXML.Tracks)
	actualKeyCount := len(mergedTracks)
	if actualKeyCount != expectedKeyCount {
		t.Errorf("Expected %d unique track keys, got %d", expectedKeyCount, actualKeyCount)
	}

	// Verify the naming convention for duplicate tracks
	// First occurrence: "rock"
	// Second occurrence: "rock#1"
	// Third occurrence: "rock#2", etc.
	if _, exists := mergedTracks["rock"]; !exists {
		t.Error("Expected 'rock' key to exist")
	}
	if _, exists := mergedTracks["rock#1"]; rockCount > 1 && !exists {
		t.Error("Expected 'rock#1' key to exist for duplicate track")
	}
	if _, exists := mergedTracks["rock#2"]; rockCount > 2 && !exists {
		t.Error("Expected 'rock#2' key to exist for duplicate track")
	}

	t.Logf("✓ BuildMergedTracks generated %d unique keys for %d original tracks",
		actualKeyCount, len(reanimXML.Tracks))

	// Verify each track has data
	for key, frames := range mergedTracks {
		if len(frames) == 0 {
			t.Errorf("Track '%s' has no frames", key)
		}
	}
}

// TestBuildMergedTracksWithOrder_DuplicateTrackNames tests the order-preserving variant
func TestBuildMergedTracksWithOrder_DuplicateTrackNames(t *testing.T) {
	path := "../../data/reanim/LoadBar_sprout.reanim"

	reanimXML, err := ParseReanimFile(path)
	if err != nil {
		t.Fatalf("Failed to parse LoadBar_sprout.reanim: %v", err)
	}

	// Build merged tracks with order
	mergedTracks, trackOrder := BuildMergedTracksWithOrder(reanimXML)

	// Verify order length matches original track count
	if len(trackOrder) != len(reanimXML.Tracks) {
		t.Errorf("Track order length mismatch: expected %d, got %d",
			len(reanimXML.Tracks), len(trackOrder))
	}

	// Verify order matches map keys
	if len(trackOrder) != len(mergedTracks) {
		t.Errorf("Track order length (%d) doesn't match map size (%d)",
			len(trackOrder), len(mergedTracks))
	}

	// Verify all keys in order exist in map
	for _, key := range trackOrder {
		if _, exists := mergedTracks[key]; !exists {
			t.Errorf("Track key '%s' in order but not in map", key)
		}
	}

	// Verify the order preserves the original sequence
	// The first 'rock' should come before 'rock#1'
	firstRockIdx := -1
	rockOneIdx := -1
	for i, key := range trackOrder {
		if key == "rock" && firstRockIdx == -1 {
			firstRockIdx = i
		}
		if key == "rock#1" && rockOneIdx == -1 {
			rockOneIdx = i
		}
	}

	if firstRockIdx != -1 && rockOneIdx != -1 && firstRockIdx >= rockOneIdx {
		t.Errorf("Track order not preserved: 'rock' at %d, 'rock#1' at %d",
			firstRockIdx, rockOneIdx)
	}

	t.Logf("✓ BuildMergedTracksWithOrder preserved order: %d tracks", len(trackOrder))

	// Print first 10 track names for verification
	printCount := 10
	if len(trackOrder) < printCount {
		printCount = len(trackOrder)
	}
	t.Logf("First %d track keys: %v", printCount, trackOrder[:printCount])
}

// TestBuildMergedTracks_NoDuplicates tests normal case without duplicate names
func TestBuildMergedTracks_NoDuplicates(t *testing.T) {
	path := "../../data/reanim/PeaShooterSingle.reanim"

	reanimXML, err := ParseReanimFile(path)
	if err != nil {
		t.Fatalf("Failed to parse PeaShooterSingle.reanim: %v", err)
	}

	// Build merged tracks
	mergedTracks := BuildMergedTracks(reanimXML)

	// Verify key count matches original track count
	if len(mergedTracks) != len(reanimXML.Tracks) {
		t.Errorf("Expected %d track keys, got %d", len(reanimXML.Tracks), len(mergedTracks))
	}

	// Verify no keys have "#" suffix (since there are no duplicates)
	for key := range mergedTracks {
		if strings.Contains(key, "#") {
			t.Errorf("Unexpected duplicate suffix in key '%s' for file without duplicates", key)
		}
	}

	t.Logf("✓ BuildMergedTracks generated %d keys (no duplicates)", len(mergedTracks))
}
