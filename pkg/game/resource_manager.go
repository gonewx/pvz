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

	"github.com/decker502/pvz/internal/particle"
	"github.com/decker502/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"gopkg.in/yaml.v3"
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
	imageCache          map[string]*ebiten.Image            // Cache for loaded images: path -> Image
	audioCache          map[string]*audio.Player            // Cache for loaded audio players: path -> Player
	audioContext        *audio.Context                      // Global audio context for audio decoding
	fontFaceCache       map[string]*text.GoTextFace         // Cache for Ebitengine v2 text faces
	reanimXMLCache      map[string]*reanim.ReanimXML        // Cache for parsed Reanim XML data: unit name -> ReanimXML
	reanimImageCache    map[string]map[string]*ebiten.Image // Cache for Reanim part images: unit name -> (image ref -> Image)
	particleConfigCache map[string]*particle.ParticleConfig // Cache for parsed particle configurations: config name -> ParticleConfig

	// YAML resource configuration
	config      *ResourceConfig   // Parsed YAML configuration
	resourceMap map[string]string // Resource ID -> file path mapping for quick lookup
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
		imageCache:          make(map[string]*ebiten.Image),
		audioCache:          make(map[string]*audio.Player),
		audioContext:        audioContext,
		fontFaceCache:       make(map[string]*text.GoTextFace),
		reanimXMLCache:      make(map[string]*reanim.ReanimXML),
		reanimImageCache:    make(map[string]map[string]*ebiten.Image),
		particleConfigCache: make(map[string]*particle.ParticleConfig),
		resourceMap:         make(map[string]string),
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

// LoadSoundEffect loads a sound effect from the specified path and caches it for future use.
// Unlike LoadAudio, this method does NOT wrap the audio in an infinite loop,
// making it suitable for one-shot sound effects like button clicks, collection sounds, etc.
// Supported formats: MP3 (.mp3) and OGG Vorbis (.ogg).
//
// Parameters:
//   - path: The file path to the sound effect resource (e.g., "assets/audio/Sound/points.ogg").
//
// Returns:
//   - A pointer to the audio player (ready to play, but not started).
//   - An error if the file cannot be opened, decoded, or the format is unsupported.
//
// Example:
//
//	player, err := rm.LoadSoundEffect("assets/audio/Sound/points.ogg")
//	if err != nil {
//	    log.Printf("Failed to load sound effect: %v", err)
//	    return err
//	}
//	player.Play() // Play once
func (rm *ResourceManager) LoadSoundEffect(path string) (*audio.Player, error) {
	// Check if the audio is already cached
	if cachedPlayer, exists := rm.audioCache[path]; exists {
		return cachedPlayer, nil
	}

	// Open the audio file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sound effect file %s: %w", path, err)
	}
	defer file.Close()

	// Read the entire file into memory
	audioData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read sound effect file %s: %w", path, err)
	}

	// Create a reader from the in-memory data
	reader := bytes.NewReader(audioData)

	// Determine the file format by extension
	ext := strings.ToLower(filepath.Ext(path))

	// Decode based on format (without infinite loop)
	var stream io.ReadSeeker

	switch ext {
	case ".mp3":
		decodedStream, err := mp3.DecodeWithoutResampling(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode MP3 sound effect %s: %w", path, err)
		}
		stream = decodedStream
	case ".ogg":
		decodedStream, err := vorbis.DecodeWithoutResampling(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode OGG sound effect %s: %w", path, err)
		}
		stream = decodedStream
	default:
		return nil, fmt.Errorf("unsupported audio format: %s (supported: .mp3, .ogg)", ext)
	}

	// Create an audio player WITHOUT infinite loop
	player, err := rm.audioContext.NewPlayer(stream)
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

// LoadReanimResources loads all Reanim resources (XML and part images) for the game.
// This method should be called once during game initialization.
//
// Returns:
//   - An error if any resource fails to load.
func (rm *ResourceManager) LoadReanimResources() error {
	// 加载植物 Reanim 资源
	// 注意：文件名使用 PascalCase（与实际文件名匹配）
	plants := []string{"PeaShooter", "SunFlower", "Wallnut", "CherryBomb"}
	for _, plantName := range plants {
		if err := rm.loadPlantReanim(plantName); err != nil {
			return fmt.Errorf("failed to load %s reanim: %w", plantName, err)
		}
	}

	// 加载僵尸 Reanim 资源
	// 注意：路障和铁桶僵尸使用基础僵尸的动画
	zombies := []string{"Zombie"}
	for _, zombieName := range zombies {
		if err := rm.loadZombieReanim(zombieName); err != nil {
			return fmt.Errorf("failed to load %s reanim: %w", zombieName, err)
		}
	}

	return nil
}

// loadPlantReanim loads Reanim resources for a specific plant.
// Parameters:
//   - name: The plant name (e.g., "peashooter", "sunflower")
//
// Returns:
//   - An error if loading fails.
func (rm *ResourceManager) loadPlantReanim(name string) error {
	// 1. 解析 .reanim 文件
	reanimPath := fmt.Sprintf("assets/effect/reanim/%s.reanim", name)
	reanimXML, err := reanim.ParseReanimFile(reanimPath)
	if err != nil {
		return fmt.Errorf("failed to parse reanim file: %w", err)
	}

	// 2. 加载部件图片
	partImages, err := rm.loadReanimPartImages(name, reanimXML, "Plants")
	if err != nil {
		return fmt.Errorf("failed to load part images: %w", err)
	}

	// 3. 存储到缓存
	rm.reanimXMLCache[name] = reanimXML
	rm.reanimImageCache[name] = partImages

	return nil
}

// loadZombieReanim loads Reanim resources for a specific zombie.
// Parameters:
//   - name: The zombie name (e.g., "zombie")
//
// Returns:
//   - An error if loading fails.
func (rm *ResourceManager) loadZombieReanim(name string) error {
	// 1. 解析 .reanim 文件
	reanimPath := fmt.Sprintf("assets/effect/reanim/%s.reanim", name)
	reanimXML, err := reanim.ParseReanimFile(reanimPath)
	if err != nil {
		return fmt.Errorf("failed to parse reanim file: %w", err)
	}

	// 2. 加载部件图片
	partImages, err := rm.loadReanimPartImages(name, reanimXML, "Zombies")
	if err != nil {
		return fmt.Errorf("failed to load part images: %w", err)
	}

	// 3. 存储到缓存
	rm.reanimXMLCache[name] = reanimXML
	rm.reanimImageCache[name] = partImages

	return nil
}

// GetReanimXML retrieves the parsed Reanim XML data for a specific unit.
// Parameters:
//   - unitName: The unit name (e.g., "peashooter", "zombie")
//
// Returns:
//   - A pointer to the ReanimXML, or nil if not found in cache.
func (rm *ResourceManager) GetReanimXML(unitName string) *reanim.ReanimXML {
	return rm.reanimXMLCache[unitName]
}

// GetReanimPartImages retrieves the part images for a specific unit.
// Parameters:
//   - unitName: The unit name (e.g., "peashooter", "zombie")
//
// Returns:
//   - A map of image reference names to images, or nil if not found in cache.
func (rm *ResourceManager) GetReanimPartImages(unitName string) map[string]*ebiten.Image {
	return rm.reanimImageCache[unitName]
}

// loadReanimPartImages loads all part images for a Reanim animation.
// Parameters:
//   - unitName: The unit name (e.g., "peashooter", "zombie")
//   - reanimXML: The parsed Reanim XML data
//   - category: The image category ("Plants" or "Zombies")
//
// Returns:
//   - A map of image reference names to images
//   - An error if any image fails to load
func (rm *ResourceManager) loadReanimPartImages(unitName string, reanimXML *reanim.ReanimXML, category string) (map[string]*ebiten.Image, error) {
	partImages := make(map[string]*ebiten.Image)

	// 收集所有需要的图片引用
	imageRefs := make(map[string]bool)
	for _, track := range reanimXML.Tracks {
		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				imageRefs[frame.ImagePath] = true
			}
		}
	}

	// 加载每个图片
	for imageRef := range imageRefs {
		// 构建图片路径
		// 例如：IMAGE_REANIM_PEASHOOTER_HEAD -> assets/images/Plants/Peashooter/Peashooter_head.png
		imagePath := rm.buildReanimImagePath(unitName, imageRef, category)

		// 加载图片
		img, err := rm.LoadImage(imagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load image %s: %w", imagePath, err)
		}

		partImages[imageRef] = img
	}

	return partImages, nil
}

// buildReanimImagePath builds the file path for a Reanim part image.
// Parameters:
//   - unitName: The unit name (e.g., "PeaShooter", "Zombie") - currently unused
//   - imageRef: The image reference name (e.g., "IMAGE_REANIM_PEASHOOTER_HEAD")
//   - category: The image category ("Plants" or "Zombies") - currently unused as all images are in assets/effect/reanim/images
//
// Returns:
//   - The constructed file path
func (rm *ResourceManager) buildReanimImagePath(unitName, imageRef, category string) string {
	// Reanim 部件图片都在 assets/reanim/ 目录下
	// 文件名格式：{unitname_lowercase}_{partname}.png
	// 例如：peashooter_head.png, zombie_arm.png
	//
	// 注意：图片引用中包含了实际的单位名称，不一定与当前加载的单位相同
	// 例如：SunFlower 的 reanim 可能引用 IMAGE_REANIM_PEASHOOTER_BACKLEAF

	// 从 imageRef 中提取完整的文件名
	// 例如：IMAGE_REANIM_PEASHOOTER_HEAD -> peashooter_head
	imageRefUpper := strings.ToUpper(imageRef)

	// 移除 IMAGE_REANIM_ 前缀（如果存在）
	imageRefUpper = strings.TrimPrefix(imageRefUpper, "IMAGE_REANIM_")

	// 转换为小写并替换下划线为文件名格式
	fileName := strings.ToLower(imageRefUpper)

	// 构建路径：assets/reanim/peashooter_head.png
	return fmt.Sprintf("assets/reanim/%s.png", fileName)
}

// LoadResourceConfig loads the resource configuration from a YAML file.
// This method should be called once during game initialization, before loading any resources.
//
// The configuration file defines resource groups and their contents, allowing resources
// to be loaded by ID instead of hard-coded paths.
//
// Parameters:
//   - configPath: Path to the YAML configuration file (e.g., "assets/config/resources.yaml")
//
// Returns:
//   - An error if the file cannot be opened or parsed
//
// Example:
//
//	rm := NewResourceManager(audioContext)
//	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
//	    log.Fatal("Failed to load resource config:", err)
//	}
func (rm *ResourceManager) LoadResourceConfig(configPath string) error {
	// Read the YAML file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read resource config %s: %w", configPath, err)
	}

	// Parse YAML into ResourceConfig struct
	var config ResourceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse resource config %s: %w", configPath, err)
	}

	// Store the parsed configuration
	rm.config = &config

	// Build resource ID -> path mapping for quick lookup
	rm.buildResourceMap()

	return nil
}

// buildResourceMap constructs a mapping from resource IDs to full file paths.
// This allows fast lookup when loading resources by ID.
//
// The mapping combines the base path with each resource's relative path.
// For example:
//
//	IMAGE_BLANK -> assets/properties/blank.png
//	SOUND_BUTTONCLICK -> assets/sounds/buttonclick.ogg
func (rm *ResourceManager) buildResourceMap() {
	if rm.config == nil {
		return
	}

	// Clear existing mapping
	rm.resourceMap = make(map[string]string)

	// Iterate through all resource groups
	for _, group := range rm.config.Groups {
		// Process images in this group
		for _, img := range group.Images {
			// Build full path: basePath + relativePath
			fullPath := buildFullPath(rm.config.BasePath, img.Path)

			// Add file extension if not present
			if filepath.Ext(fullPath) == "" {
				fullPath += ".png" // Default to PNG for images
			}

			rm.resourceMap[img.ID] = fullPath
		}

		// Process sounds in this group
		for _, sound := range group.Sounds {
			fullPath := buildFullPath(rm.config.BasePath, sound.Path)

			// Add file extension if not present
			if filepath.Ext(fullPath) == "" {
				fullPath += ".ogg" // Default to OGG for sounds
			}

			rm.resourceMap[sound.ID] = fullPath
		}

		// Process fonts in this group
		for _, font := range group.Fonts {
			fullPath := buildFullPath(rm.config.BasePath, font.Path)
			rm.resourceMap[font.ID] = fullPath
		}
	}
}

// LoadImageByID loads an image resource using its resource ID.
// The resource ID must be defined in the YAML configuration file.
//
// This method first looks up the file path associated with the ID,
// then loads the image using the standard LoadImage method.
//
// Parameters:
//   - resourceID: The resource ID (e.g., "IMAGE_BLANK", "IMAGE_REANIM_SEEDS")
//
// Returns:
//   - A pointer to the loaded ebiten.Image
//   - An error if the ID is not found or the image cannot be loaded
//
// Example:
//
//	img, err := rm.LoadImageByID("IMAGE_BACKGROUND1")
//	if err != nil {
//	    log.Printf("Failed to load image: %v", err)
//	}
func (rm *ResourceManager) LoadImageByID(resourceID string) (*ebiten.Image, error) {
	// Check if resource config is loaded
	if rm.config == nil {
		return nil, fmt.Errorf("resource config not loaded - call LoadResourceConfig first")
	}

	// Look up the file path for this resource ID
	filePath, exists := rm.resourceMap[resourceID]
	if !exists {
		return nil, fmt.Errorf("resource ID not found: %s", resourceID)
	}

	// Load the image using the resolved path
	return rm.LoadImage(filePath)
}

// GetImageByID retrieves a previously loaded image using its resource ID.
// If the image has not been loaded yet, it returns nil.
//
// Parameters:
//   - resourceID: The resource ID (e.g., "IMAGE_BLANK")
//
// Returns:
//   - A pointer to the cached ebiten.Image, or nil if not found
//
// Example:
//
//	img := rm.GetImageByID("IMAGE_BACKGROUND1")
//	if img == nil {
//	    // Image not loaded yet
//	}
func (rm *ResourceManager) GetImageByID(resourceID string) *ebiten.Image {
	if rm.config == nil {
		return nil
	}

	// Look up the file path
	filePath, exists := rm.resourceMap[resourceID]
	if !exists {
		return nil
	}

	// Get from cache
	return rm.GetImage(filePath)
}

// LoadResourceGroup loads all resources in a specified group.
// Resource groups are defined in the YAML configuration file.
//
// This is useful for batch-loading related resources, such as:
//   - "init" - Initial resources needed at startup
//   - "loadingimages" - Resources for the loading screen
//   - "delayload_background1" - Resources for a specific level
//
// Parameters:
//   - groupName: The name of the resource group (e.g., "init", "loadingimages")
//
// Returns:
//   - An error if the group is not found or any resource fails to load
//
// Example:
//
//	// Load all initial resources at game startup
//	if err := rm.LoadResourceGroup("init"); err != nil {
//	    log.Fatal("Failed to load init resources:", err)
//	}
func (rm *ResourceManager) LoadResourceGroup(groupName string) error {
	// Check if resource config is loaded
	if rm.config == nil {
		return fmt.Errorf("resource config not loaded - call LoadResourceConfig first")
	}

	// Find the resource group
	group, exists := rm.config.Groups[groupName]
	if !exists {
		return fmt.Errorf("resource group not found: %s", groupName)
	}

	// Load all images in the group
	for _, img := range group.Images {
		if _, err := rm.LoadImageByID(img.ID); err != nil {
			return fmt.Errorf("failed to load image %s in group %s: %w", img.ID, groupName, err)
		}
	}

	// Load all sounds in the group
	for _, sound := range group.Sounds {
		// Look up the file path
		filePath, exists := rm.resourceMap[sound.ID]
		if !exists {
			return fmt.Errorf("sound resource ID not found: %s", sound.ID)
		}

		// Load the sound effect (use LoadSoundEffect for non-looping sounds)
		if _, err := rm.LoadSoundEffect(filePath); err != nil {
			return fmt.Errorf("failed to load sound %s in group %s: %w", sound.ID, groupName, err)
		}
	}

	// Fonts are not loaded here as they require a size parameter
	// They should be loaded individually using LoadFont when needed

	return nil
}

// LoadParticleConfig loads a particle configuration file from the particles directory and caches it.
// If the configuration has already been loaded, it returns the cached version.
//
// Parameters:
//   - name: The particle configuration name. Can be either:
//   - A simple name (e.g., "Award", "BossExplosion") - will be loaded from assets/effect/particles/
//   - A path with directory separators (e.g., "path/to/config") - will be used as-is
//
// Returns:
//   - A pointer to the loaded ParticleConfig
//   - An error if the file cannot be loaded or parsed
//
// Example usage:
//
//	config, err := rm.LoadParticleConfig("Award")
//	if err != nil {
//	    log.Printf("Failed to load particle config: %v", err)
//	}
//	fmt.Printf("Loaded %d emitters\n", len(config.Emitters))
func (rm *ResourceManager) LoadParticleConfig(name string) (*particle.ParticleConfig, error) {
	// Check cache first
	if config, exists := rm.particleConfigCache[name]; exists {
		return config, nil
	}

	// Construct file path
	var path string
	if filepath.Dir(name) != "." {
		// Name contains path separators, use as-is
		path = name + ".xml"
	} else {
		// Simple name, add default prefix
		path = filepath.Join("assets/effect/particles", name+".xml")
	}

	// Parse the configuration
	config, err := particle.ParseParticleXML(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load particle config %s: %w", name, err)
	}

	// Cache and return
	rm.particleConfigCache[name] = config
	return config, nil
}

// GetParticleConfig retrieves a cached particle configuration by name.
// Returns nil if the configuration has not been loaded yet.
//
// Parameters:
//   - name: The particle configuration name (e.g., "Award", "BossExplosion")
//
// Returns:
//   - A pointer to the ParticleConfig, or nil if not found in cache
//
// Example usage:
//
//	config := rm.GetParticleConfig("Award")
//	if config == nil {
//	    log.Println("Config not loaded yet")
//	}
func (rm *ResourceManager) GetParticleConfig(name string) *particle.ParticleConfig {
	return rm.particleConfigCache[name]
}
