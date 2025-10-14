package game

// ResourceConfig represents the top-level resource configuration loaded from YAML.
// It defines the structure of assets/config/resources.yaml.
//
// Structure:
//
//	version: "1.0"
//	base_path: assets
//	groups:
//	  group_name:
//	    images: [...]
//	    sounds: [...]
//	    fonts: [...]
type ResourceConfig struct {
	Version  string                   `yaml:"version"`   // Configuration file version
	BasePath string                   `yaml:"base_path"` // Base path for all resources (e.g., "assets")
	Groups   map[string]ResourceGroup `yaml:"groups"`    // Resource groups keyed by group name
}

// ResourceGroup represents a collection of related resources that can be loaded together.
// Each group contains lists of images, sounds, and fonts.
//
// Example from resources.yaml:
//
//	init:
//	  images:
//	    - id: IMAGE_BLANK
//	      path: properties/blank
type ResourceGroup struct {
	Images []ImageResource `yaml:"images"` // List of image resources in this group
	Sounds []SoundResource `yaml:"sounds"` // List of sound resources in this group
	Fonts  []FontResource  `yaml:"fonts"`  // List of font resources in this group
}

// ImageResource represents a single image resource definition.
// It can be a simple image or a sprite sheet with rows/cols.
//
// Fields:
//   - ID: Unique identifier for the image (e.g., "IMAGE_BLANK", "IMAGE_REANIM_SEEDS")
//   - Path: Relative path from base_path to the image file (without extension)
//   - Cols: Number of columns in sprite sheet (optional, for sprite sheets)
//   - Rows: Number of rows in sprite sheet (optional, for sprite sheets)
//   - Attrs: Additional attributes like alphagrid (optional)
//
// Examples:
//
//	Simple image:
//	  - id: IMAGE_BLANK
//	    path: properties/blank
//
//	Sprite sheet:
//	  - id: IMAGE_REANIM_SEEDS
//	    path: reanim/seeds.png
//	    cols: 9
//
//	With attributes:
//	  - id: IMAGE_ZOMBIE_NOTE1
//	    path: images/ZombieNoteBlack1.png
//	    attrs:
//	      alphagrid: ZombieNote1
type ImageResource struct {
	ID    string                 `yaml:"id"`              // Resource ID (unique identifier)
	Path  string                 `yaml:"path"`            // Relative file path from base_path
	Cols  int                    `yaml:"cols,omitempty"`  // Sprite sheet columns (0 if not a sprite sheet)
	Rows  int                    `yaml:"rows,omitempty"`  // Sprite sheet rows (0 if not a sprite sheet)
	Attrs map[string]interface{} `yaml:"attrs,omitempty"` // Additional attributes (e.g., alphagrid)
}

// SoundResource represents a single sound/audio resource definition.
//
// Fields:
//   - ID: Unique identifier for the sound (e.g., "SOUND_BUTTONCLICK")
//   - Path: Relative path from base_path to the audio file
//
// Example:
//   - id: SOUND_BUTTONCLICK
//     path: sounds/buttonclick.ogg
type SoundResource struct {
	ID   string `yaml:"id"`   // Resource ID (unique identifier)
	Path string `yaml:"path"` // Relative file path from base_path
}

// FontResource represents a single font resource definition.
//
// Fields:
//   - ID: Unique identifier for the font (e.g., "FONT_HOUSEOFTERROR28")
//   - Path: Relative path from base_path to the font file
//
// Example:
//   - id: FONT_HOUSEOFTERROR28
//     path: data/HouseofTerror28.txt
type FontResource struct {
	ID   string `yaml:"id"`   // Resource ID (unique identifier)
	Path string `yaml:"path"` // Relative file path from base_path
}

// buildFullPath constructs the full file path for a resource.
// It combines the base path with the resource's relative path.
//
// Parameters:
//   - basePath: The base path from ResourceConfig (e.g., "assets")
//   - relativePath: The resource's relative path (e.g., "images/background1.png")
//
// Returns:
//   - The full file path (e.g., "assets/images/background1.png")
//
// Note: This is a helper function that may be used by ResourceManager.
func buildFullPath(basePath, relativePath string) string {
	if basePath == "" {
		return relativePath
	}
	// Simple path joining - handles the case where relative path might start with /
	if len(relativePath) > 0 && relativePath[0] == '/' {
		return basePath + relativePath
	}
	return basePath + "/" + relativePath
}
