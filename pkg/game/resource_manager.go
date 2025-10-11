package game

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ResourceManager is responsible for centralized management of game resources.
// It provides loading and caching mechanisms for images and audio assets,
// ensuring that resources are loaded only once and reused throughout the game.
//
// The ResourceManager implements the following key features:
// - Image loading and caching (PNG format support)
// - Audio loading and caching (MP3/OGG format support)
// - Error handling for missing or corrupted resources
// - Resource path normalization
//
// Thread Safety Note:
// This implementation is NOT thread-safe. The internal caches use standard Go maps,
// which are not safe for concurrent access. If you need to load resources from
// multiple goroutines, you must:
//   - Synchronize access using external locking (e.g., sync.RWMutex), OR
//   - Pre-load all resources in the main goroutine before starting concurrent operations
//
// For the current single-threaded game loop, no synchronization is needed.
//
// Usage:
//
//	audioContext := audio.NewContext(48000)
//	rm := NewResourceManager(audioContext)
//	img, err := rm.LoadImage("assets/images/interface/MainMenu.png")
//	if err != nil {
//	    log.Printf("Failed to load image: %v", err)
//	}
type ResourceManager struct {
	imageCache    map[string]*ebiten.Image     // Cache for loaded images: path -> Image
	audioCache    map[string]*audio.Player     // Cache for loaded audio players: path -> Player
	audioContext  *audio.Context               // Global audio context for audio decoding
	fontFaceCache map[string]*text.GoTextFace  // Cache for Ebitengine v2 text faces
}

// NewResourceManager creates and initializes a new ResourceManager instance.
// The audioContext parameter is required for audio decoding and playback.
// It should be created once at game startup with a sample rate of 48000 Hz.
//
// Parameters:
//   - audioContext: The global audio context used for decoding and playing audio files.
//
// Returns:
//   - A pointer to a newly initialized ResourceManager with empty caches.
//
// Example:
//
//	audioContext := audio.NewContext(48000)
//	resourceManager := NewResourceManager(audioContext)
func NewResourceManager(audioContext *audio.Context) *ResourceManager {
	return &ResourceManager{
		imageCache:    make(map[string]*ebiten.Image),
		audioCache:    make(map[string]*audio.Player),
		audioContext:  audioContext,
		fontFaceCache: make(map[string]*text.GoTextFace),
	}
}

// LoadImage loads an image file from the specified path and caches it for future use.
// If the image has already been loaded, it returns the cached version.
// Supported formats: PNG (via image/png decoder).
//
// Parameters:
//   - path: The file path to the image resource (e.g., "assets/images/interface/MainMenu.png").
//
// Returns:
//   - A pointer to the loaded ebiten.Image.
//   - An error if the file cannot be opened, decoded, or converted.
//
// Error handling:
//   - Returns an error if the file does not exist or cannot be opened.
//   - Returns an error if the image format is not supported or the file is corrupted.
//   - Does not panic - all errors are returned to the caller for handling.
//
// Example:
//
//	img, err := rm.LoadImage("assets/images/interface/MainMenu.png")
//	if err != nil {
//	    log.Printf("Failed to load image: %v", err)
//	    return err
//	}
func (rm *ResourceManager) LoadImage(path string) (*ebiten.Image, error) {
	// Check if the image is already cached
	if cachedImage, exists := rm.imageCache[path]; exists {
		return cachedImage, nil
	}

	// Open the image file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file %s: %w", path, err)
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", path, err)
	}

	// Convert to Ebitengine image
	ebitenImg := ebiten.NewImageFromImage(img)

	// Store in cache
	rm.imageCache[path] = ebitenImg

	return ebitenImg, nil
}

// GetImage retrieves a previously loaded image from the cache.
// If the image has not been loaded yet, it returns nil.
// Use LoadImage to load and cache an image before calling this method.
//
// Parameters:
//   - path: The file path of the image resource.
//
// Returns:
//   - A pointer to the cached ebiten.Image, or nil if not found in cache.
//
// Example:
//
//	img := rm.GetImage("assets/images/interface/MainMenu.png")
//	if img == nil {
//	    // Image not loaded yet, need to call LoadImage first
//	}
func (rm *ResourceManager) GetImage(path string) *ebiten.Image {
	return rm.imageCache[path]
}

// LoadAudio loads an audio file from the specified path and caches it for future use.
// If the audio has already been loaded, it returns the cached player.
// Supported formats: MP3 (.mp3) and OGG Vorbis (.ogg).
//
// The audio is automatically wrapped in an infinite loop, making it suitable for background music.
// For sound effects that should not loop, consider adding a separate LoadSoundEffect method.
//
// Parameters:
//   - path: The file path to the audio resource (e.g., "assets/audio/Music/mainmenubgm.mp3").
//
// Returns:
//   - A pointer to the audio player (ready to play, but not started).
//   - An error if the file cannot be opened, decoded, or the format is unsupported.
//
// Error handling:
//   - Returns an error if the file does not exist or cannot be opened.
//   - Returns an error if the audio format is not supported (must be .mp3 or .ogg).
//   - Returns an error if the file is corrupted or cannot be decoded.
//   - Does not panic - all errors are returned to the caller for handling.
//
// Example:
//
//	player, err := rm.LoadAudio("assets/audio/Music/mainmenubgm.mp3")
//	if err != nil {
//	    log.Printf("Failed to load audio: %v", err)
//	    return err
//	}
//	player.Play() // Start playing the music
func (rm *ResourceManager) LoadAudio(path string) (*audio.Player, error) {
	// Check if the audio is already cached
	if cachedPlayer, exists := rm.audioCache[path]; exists {
		return cachedPlayer, nil
	}

	// Open the audio file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file %s: %w", path, err)
	}
	defer file.Close()

	// Read the entire file into memory to avoid file handle issues
	// This allows the audio stream to seek without keeping the file open
	audioData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file %s: %w", path, err)
	}

	// Create a reader from the in-memory data
	reader := bytes.NewReader(audioData)

	// Determine the file format by extension
	ext := strings.ToLower(filepath.Ext(path))

	// Decode based on format
	var stream interface {
		io.ReadSeeker
		Length() int64
	}

	switch ext {
	case ".mp3":
		decodedStream, err := mp3.DecodeWithoutResampling(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode MP3 audio %s: %w", path, err)
		}
		stream = decodedStream
	case ".ogg":
		decodedStream, err := vorbis.DecodeWithoutResampling(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode OGG audio %s: %w", path, err)
		}
		stream = decodedStream
	default:
		return nil, fmt.Errorf("unsupported audio format: %s (supported: .mp3, .ogg)", ext)
	}

	// Wrap the stream in an infinite loop for background music
	loopStream := audio.NewInfiniteLoop(stream, stream.Length())

	// Create an audio player
	player, err := rm.audioContext.NewPlayer(loopStream)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio player for %s: %w", path, err)
	}

	// Store in cache
	rm.audioCache[path] = player

	return player, nil
}

// GetAudioPlayer retrieves a previously loaded audio player from the cache.
// If the audio has not been loaded yet, it returns nil.
// Use LoadAudio to load and cache an audio player before calling this method.
//
// Parameters:
//   - path: The file path of the audio resource.
//
// Returns:
//   - A pointer to the cached audio.Player, or nil if not found in cache.
//
// Example:
//
//	player := rm.GetAudioPlayer("assets/audio/Music/mainmenubgm.mp3")
//	if player == nil {
//	    // Audio not loaded yet, need to call LoadAudio first
//	}
func (rm *ResourceManager) GetAudioPlayer(path string) *audio.Player {
	return rm.audioCache[path]
}

// LoadFont loads a TrueType/OpenType font from the specified path and creates a text face with the given size.
// The font face is cached for future use with a cache key combining path and size.
// Supported formats: .ttf, .otf
//
// Parameters:
//   - path: The file path to the font resource (e.g., "assets/fonts/briannetod.ttf").
//   - size: The font size in pixels.
//
// Returns:
//   - A pointer to the text.GoTextFace ready for rendering.
//   - An error if the file cannot be opened or parsed.
//
// Example:
//
//	fontFace, err := rm.LoadFont("assets/fonts/briannetod.ttf", 32)
//	if err != nil {
//	    log.Printf("Failed to load font: %v", err)
//	    return err
//	}
func (rm *ResourceManager) LoadFont(path string, size float64) (*text.GoTextFace, error) {
	// Create cache key combining path and size
	cacheKey := fmt.Sprintf("%s:%.1f", path, size)

	// Check if the font face is already cached
	if cachedFace, exists := rm.fontFaceCache[cacheKey]; exists {
		return cachedFace, nil
	}

	// Read font file
	fontData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read font file %s: %w", path, err)
	}

	// Create GoTextFaceSource from font data
	source, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		return nil, fmt.Errorf("failed to create font source for %s: %w", path, err)
	}

	// Create GoTextFace with specified size
	goTextFace := &text.GoTextFace{
		Source:    source,
		Size:      size,
		Direction: text.DirectionLeftToRight,
	}

	// Store in cache
	rm.fontFaceCache[cacheKey] = goTextFace

	return goTextFace, nil
}

// GetFont retrieves a previously loaded font face from the cache.
// If the font has not been loaded yet, it returns nil.
// Use LoadFont to load and cache a font before calling this method.
//
// Parameters:
//   - path: The file path of the font resource.
//   - size: The font size in points.
//
// Returns:
//   - A pointer to the cached text.GoTextFace, or nil if not found in cache.
func (rm *ResourceManager) GetFont(path string, size float64) *text.GoTextFace {
	cacheKey := fmt.Sprintf("%s:%.1f", path, size)
	return rm.fontFaceCache[cacheKey]
}
