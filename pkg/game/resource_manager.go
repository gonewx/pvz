package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // Register GIF decoder
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"io"
	"log"
	"path/filepath"
	"strings"

	auaudio "github.com/gonewx/pvz/internal/audio"
	"github.com/gonewx/pvz/internal/particle"
	"github.com/gonewx/pvz/internal/reanim"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/embedded"
	"github.com/gonewx/pvz/pkg/utils"
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
// - Image loading and caching (PNG, JPEG, GIF format support)
// - Audio loading and caching (MP3, OGG, AU format support)
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
	bitmapFontCache     map[string]*utils.BitmapFont        // Cache for bitmap fonts (Story 8.2)
	reanimXMLCache      map[string]*reanim.ReanimXML        // Cache for parsed Reanim XML data: unit name -> ReanimXML
	reanimImageCache    map[string]map[string]*ebiten.Image // Cache for Reanim part images: unit name -> (image ref -> Image)
	particleConfigCache map[string]*particle.ParticleConfig // Cache for parsed particle configurations: config name -> ParticleConfig

	// Story 13.6: Reanim 配置管理器
	reanimConfigManager *config.ReanimConfigManager // Reanim 配置管理器（用于配置驱动的动画播放）

	// Story 8.8: 背景音乐管理（用于淡出功能）
	currentBGMPlayer *audio.Player // 当前播放的背景音乐播放器
	bgmFadeOut       bool          // 是否正在淡出
	bgmFadeDuration  float64       // 淡出总时长（秒）
	bgmFadeElapsed   float64       // 淡出已经过时间（秒）
	bgmInitialVolume float64       // 淡出开始时的音量

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
		bitmapFontCache:     make(map[string]*utils.BitmapFont),
		reanimXMLCache:      make(map[string]*reanim.ReanimXML),
		reanimImageCache:    make(map[string]map[string]*ebiten.Image),
		particleConfigCache: make(map[string]*particle.ParticleConfig),
		resourceMap:         make(map[string]string),
	}
}

// LoadImage loads an image file from the specified path and caches it for future use.
// If the image has already been loaded, it returns the cached version.
// Supported formats: PNG, JPEG, GIF (via image decoders).
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

	// Open the image file from embedded FS
	file, err := embedded.Open(path)
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

// LoadImageWithAlphaMask loads a color image and a separate alpha mask, then composites them
// into a single RGBA image. This is used for textures that store RGB and Alpha separately
// to save space (e.g., sod1row.jpg + sod1row_.png).
//
// The alpha mask is expected to be a grayscale image where:
//   - White (255) = fully opaque
//   - Black (0) = fully transparent
//
// Parameters:
//   - rgbPath: Path to the RGB color image (typically JPEG)
//   - alphaPath: Path to the alpha mask image (typically PNG grayscale)
//
// Returns:
//   - A composited RGBA ebiten.Image with transparency applied
//   - An error if loading or compositing fails
//
// Example:
//
//	img, err := rm.LoadImageWithAlphaMask(
//	    "assets/images/sod1row.jpg",
//	    "assets/images/sod1row_.png")
func (rm *ResourceManager) LoadImageWithAlphaMask(rgbPath, alphaPath string) (*ebiten.Image, error) {
	// Create cache key for the composite image
	cacheKey := rgbPath + "+" + alphaPath

	// Check if already cached
	if cachedImage, exists := rm.imageCache[cacheKey]; exists {
		return cachedImage, nil
	}

	// Load RGB image from embedded FS
	rgbFile, err := embedded.Open(rgbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open RGB image %s: %w", rgbPath, err)
	}
	defer rgbFile.Close()

	rgbImg, _, err := image.Decode(rgbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode RGB image %s: %w", rgbPath, err)
	}

	// Load alpha mask image from embedded FS
	alphaFile, err := embedded.Open(alphaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open alpha mask %s: %w", alphaPath, err)
	}
	defer alphaFile.Close()

	alphaMask, _, err := image.Decode(alphaFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode alpha mask %s: %w", alphaPath, err)
	}

	// Verify dimensions match
	bounds := rgbImg.Bounds()
	alphaBounds := alphaMask.Bounds()
	if bounds.Dx() != alphaBounds.Dx() || bounds.Dy() != alphaBounds.Dy() {
		return nil, fmt.Errorf("RGB image size %v does not match alpha mask size %v", bounds, alphaBounds)
	}

	// Create RGBA image for compositing
	rgba := image.NewRGBA(bounds)

	// Composite RGB + Alpha
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get RGB color
			r, g, b, _ := rgbImg.At(x, y).RGBA()

			// Get alpha from grayscale value
			gray, _, _, _ := alphaMask.At(x, y).RGBA()
			alpha := uint8(gray >> 8) // Convert 16-bit to 8-bit

			// Set RGBA pixel
			rgba.Set(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: alpha,
			})
		}
	}

	// Convert to Ebitengine image
	ebitenImg := ebiten.NewImageFromImage(rgba)

	// Store in cache
	rm.imageCache[cacheKey] = ebitenImg

	log.Printf("✅ Composited RGBA image: %s + %s -> %dx%d", rgbPath, alphaPath, bounds.Dx(), bounds.Dy())

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
// Supported formats: MP3 (.mp3), OGG Vorbis (.ogg), and Sun/NeXT audio (.au).
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
//   - Returns an error if the audio format is not supported (must be .mp3, .ogg, or .au).
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

	// Read the audio file from embedded FS
	audioData, err := embedded.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file %s: %w", path, err)
	}

	// Create a reader from the in-memory data
	reader := bytes.NewReader(audioData)

	// Determine the file format by extension
	ext := strings.ToLower(filepath.Ext(path))

	// Decode based on format (with resampling to match AudioContext sample rate)
	var stream interface {
		io.ReadSeeker
		Length() int64
	}

	switch ext {
	case ".mp3":
		decodedStream, err := mp3.DecodeWithSampleRate(rm.audioContext.SampleRate(), reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode MP3 audio %s: %w", path, err)
		}
		stream = decodedStream
	case ".ogg":
		decodedStream, err := vorbis.DecodeWithSampleRate(rm.audioContext.SampleRate(), reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode OGG audio %s: %w", path, err)
		}
		stream = decodedStream
	case ".au":
		decodedStream, err := auaudio.DecodeAU(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode AU audio %s: %w", path, err)
		}
		stream = decodedStream
	default:
		return nil, fmt.Errorf("unsupported audio format: %s (supported: .mp3, .ogg, .au)", ext)
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
// Supported formats: MP3 (.mp3), OGG Vorbis (.ogg), and Sun/NeXT audio (.au).
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

	// Read the sound effect file from embedded FS
	audioData, err := embedded.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read sound effect file %s: %w", path, err)
	}

	// Create a reader from the in-memory data
	reader := bytes.NewReader(audioData)

	// Determine the file format by extension
	ext := strings.ToLower(filepath.Ext(path))

	// Decode based on format (with resampling to match AudioContext sample rate)
	var stream io.ReadSeeker

	switch ext {
	case ".mp3":
		decodedStream, err := mp3.DecodeWithSampleRate(rm.audioContext.SampleRate(), reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode MP3 sound effect %s: %w", path, err)
		}
		stream = decodedStream
	case ".ogg":
		decodedStream, err := vorbis.DecodeWithSampleRate(rm.audioContext.SampleRate(), reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode OGG sound effect %s: %w", path, err)
		}
		stream = decodedStream
	case ".au":
		decodedStream, err := auaudio.DecodeAU(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode AU sound effect %s: %w", path, err)
		}
		stream = decodedStream
	default:
		return nil, fmt.Errorf("unsupported audio format: %s (supported: .mp3, .ogg, .au)", ext)
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

// GetAudioPlayerByID retrieves a previously loaded audio player using its resource ID.
// If the audio has not been loaded yet, it returns nil.
//
// Parameters:
//   - resourceID: The resource ID (e.g., "SOUND_SCREAM", "SOUND_CHOMP")
//
// Returns:
//   - A pointer to the cached audio.Player, or nil if not found
func (rm *ResourceManager) GetAudioPlayerByID(resourceID string) *audio.Player {
	if rm.config == nil {
		return nil
	}

	// Look up the file path
	filePath, exists := rm.resourceMap[resourceID]
	if !exists {
		return nil
	}

	// Get from cache
	return rm.audioCache[filePath]
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

	// Read font file from embedded FS
	fontData, err := embedded.ReadFile(path)
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
	// 自动扫描并加载 data/reanim 目录下的所有 .reanim 文件（Epic 13 迁移）
	// 优点：
	// - 无需手动维护加载列表
	// - 自动发现新添加的动画文件
	// - 避免遗漏资源（如之前的 SelectorScreen.reanim）

	reanimDir := "data/reanim"
	pattern := reanimDir + "/*.reanim"
	files, err := embedded.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to scan reanim directory: %w", err)
	}

	log.Printf("[ResourceManager] 发现 %d 个 reanim 文件，开始加载...", len(files))

	successCount := 0
	failedFiles := []string{}

	for _, filePath := range files {
		// 提取文件名（去掉路径和 .reanim 扩展名）
		fileName := filepath.Base(filePath)
		reanimName := strings.TrimSuffix(fileName, ".reanim") // 安全地去掉 ".reanim" 扩展名

		// 加载 reanim 资源
		if err := rm.loadReanimAuto(reanimName, filePath); err != nil {
			log.Printf("[ResourceManager] ⚠️  跳过 %s: %v", reanimName, err)
			failedFiles = append(failedFiles, reanimName)
			continue
		}

		successCount++
	}

	log.Printf("[ResourceManager] ✅ 成功加载 %d/%d 个 reanim 资源", successCount, len(files))
	if len(failedFiles) > 0 {
		log.Printf("[ResourceManager] ⚠️  失败 %d 个: %v", len(failedFiles), failedFiles)
	}

	return nil
}

// loadReanimAuto 自动加载任意 reanim 资源（植物、僵尸、特效等）
// 这是一个统一的加载函数，取代了之前的 loadPlantReanim、loadZombieReanim、loadEffectReanim
//
// Parameters:
//   - name: reanim 名称（如 "PeaShooterSingle", "Zombie", "SelectorScreen"）
//   - filePath: reanim 文件的完整路径
//
// Returns:
//   - 加载失败时返回 error
func (rm *ResourceManager) loadReanimAuto(name string, filePath string) error {
	// 1. 解析 .reanim 文件
	reanimXML, err := reanim.ParseReanimFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse reanim file: %w", err)
	}

	// 2. 加载部件图片（所有图片统一从资源配置加载）
	partImages, err := rm.loadReanimPartImages(name, reanimXML, "")
	if err != nil {
		return fmt.Errorf("failed to load part images: %w", err)
	}

	// 3. 存储到缓存
	rm.reanimXMLCache[name] = reanimXML
	rm.reanimImageCache[name] = partImages

	log.Printf("[ResourceManager] ✅ 已加载 %s (FPS=%d, Tracks=%d, Images=%d)",
		name, reanimXML.FPS, len(reanimXML.Tracks), len(partImages))

	return nil
}

// loadPlantReanim loads Reanim resources for a specific plant.
// 已废弃：请使用 loadReanimAuto
// Parameters:
//   - name: The plant name (e.g., "peashooter", "sunflower")
//
// Returns:
//   - An error if loading fails.
func (rm *ResourceManager) loadPlantReanim(name string) error {
	// 1. 解析 .reanim 文件
	reanimPath := fmt.Sprintf("data/reanim/%s.reanim", name)
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
	reanimPath := fmt.Sprintf("data/reanim/%s.reanim", name)
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

// loadEffectReanim loads Reanim resources for special effects (e.g., Sun, explosions).
// Parameters:
//   - name: The effect name (e.g., "Sun")
//
// Returns:
//   - An error if loading fails.
//
// Story 8.2 QA修复：阳光动画使用通用的特效加载逻辑，
// 从 assets/reanim/ 目录加载部件图片（而非 Plants 或 Zombies 子目录）
func (rm *ResourceManager) loadEffectReanim(name string) error {
	// 1. 解析 .reanim 文件
	reanimPath := fmt.Sprintf("data/reanim/%s.reanim", name)
	log.Printf("[ResourceManager] Loading effect reanim: %s from %s", name, reanimPath)
	reanimXML, err := reanim.ParseReanimFile(reanimPath)
	if err != nil {
		return fmt.Errorf("failed to parse reanim file: %w", err)
	}
	log.Printf("[ResourceManager] Parsed %s.reanim: FPS=%d, Tracks=%d", name, reanimXML.FPS, len(reanimXML.Tracks))

	// 2. 加载部件图片（从 assets/reanim/ 目录）
	partImages, err := rm.loadReanimPartImages(name, reanimXML, "")
	if err != nil {
		return fmt.Errorf("failed to load part images: %w", err)
	}
	log.Printf("[ResourceManager] Loaded %d part images for %s", len(partImages), name)
	for imgID := range partImages {
		log.Printf("[ResourceManager]   - %s", imgID)
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

	// Special handling for SelectorScreen background images (jpg + png overlay)
	// These background images need compositing to remove black backgrounds
	needsCompositing := map[string]string{
		"IMAGE_REANIM_SELECTORSCREEN_BG_CENTER": "IMAGE_REANIM_SELECTORSCREEN_BG_CENTER_OVERLAY",
		"IMAGE_REANIM_SELECTORSCREEN_BG_LEFT":   "IMAGE_REANIM_SELECTORSCREEN_BG_LEFT_OVERLAY",
		"IMAGE_REANIM_SELECTORSCREEN_BG_RIGHT":  "IMAGE_REANIM_SELECTORSCREEN_BG_RIGHT_OVERLAY",
		"IMAGE_REANIM_ZOMBIESWON":               "IMAGE_REANIM_ZOMBIESWON_OVERLAY",
		"IMAGE_REANIM_CRAZYDAVE_BODY1":          "IMAGE_REANIM_CRAZYDAVE_BODY1_MASK",
	}

	// 加载每个图片
	for imageRef := range imageRefs {
		var img *ebiten.Image
		var err error

		// Check if this image needs compositing (jpg base + png overlay)
		if overlayID, needsComposite := needsCompositing[imageRef]; needsComposite {
			// Load composited image (jpg + png mask)
			log.Printf("[ResourceManager] Applying alpha mask to %s using %s", imageRef, overlayID)
			img, err = rm.LoadCompositedImage(imageRef, overlayID)
			if err != nil {
				log.Printf("[ResourceManager] Warning: Failed to composite %s, falling back to base image: %v", imageRef, err)
				// Fallback: load base image only
				img, err = rm.LoadImageByID(imageRef)
			} else {
				log.Printf("[ResourceManager] ✅ Successfully applied alpha mask to %s", imageRef)
			}
		} else {
			// 直接使用资源 ID 从配置文件加载图片
			// 例如：IMAGE_REANIM_PEASHOOTER_HEAD
			// 资源配置文件中定义了正确的路径：reanim/PeaShooter_head.png
			img, err = rm.LoadImageByID(imageRef)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to load image %s: %w", imageRef, err)
		}

		partImages[imageRef] = img
	}

	return partImages, nil
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
	// Read the YAML file from embedded FS
	data, err := embedded.ReadFile(configPath)
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
			// Try to find the actual file with common image extensions
			if filepath.Ext(fullPath) == "" {
				// Try common image extensions in order of preference
				extensions := []string{".png", ".jpg", ".jpeg", ".gif"}
				found := false
				for _, ext := range extensions {
					testPath := fullPath + ext
					if embedded.Exists(testPath) {
						fullPath = testPath
						found = true
						break
					}
				}
				if !found {
					// Default to PNG if file not found
					fullPath += ".png"
				}
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
	return rm.imageCache[filePath]
}

// GetShadowImage 获取阴影贴图
// 如果尚未加载,则自动加载 plantshadow.png
// 此方法确保阴影贴图只加载一次并被缓存
//
// 返回值:
//   - 阴影贴图的 ebiten.Image 指针
//   - 如果加载失败则返回 nil
//
// 用法:
//
//	shadowImg := rm.GetShadowImage()
//	if shadowImg != nil {
//	    screen.DrawImage(shadowImg, op)
//	}
func (rm *ResourceManager) GetShadowImage() *ebiten.Image {
	shadowPath := "assets/images/plantshadow.png"

	// 检查缓存
	if cachedImg, exists := rm.imageCache[shadowPath]; exists {
		return cachedImg
	}

	// 加载阴影贴图
	img, err := rm.LoadImage(shadowPath)
	if err != nil {
		log.Printf("警告: 无法加载阴影贴图 %s: %v", shadowPath, err)
		return nil
	}

	return img
}

// LoadCompositedImage loads a base image and its alpha mask, then applies the mask to remove background.
// This is used for PVZ assets where a JPG base image (with black background) needs to be
// combined with a PNG mask to create transparency.
//
// The PNG mask works as follows:
//   - White pixels in mask = fully opaque (keep the JPG pixel)
//   - Black pixels in mask = fully transparent (remove background)
//   - Gray pixels = partial transparency
//
// Parameters:
//   - baseResourceID: Resource ID of the base image (e.g., "IMAGE_REANIM_SELECTORSCREEN_BG_CENTER")
//   - maskResourceID: Resource ID of the alpha mask (e.g., "IMAGE_REANIM_SELECTORSCREEN_BG_CENTER_OVERLAY")
//
// Returns:
//   - A new image with mask applied, or error if loading fails
//
// Example:
//
//	bgCenter, err := rm.LoadCompositedImage(
//	    "IMAGE_REANIM_SELECTORSCREEN_BG_CENTER",
//	    "IMAGE_REANIM_SELECTORSCREEN_BG_CENTER_OVERLAY")
//
// Note: The composited image is NOT cached. If you need to reuse it, store it yourself.
func (rm *ResourceManager) LoadCompositedImage(baseResourceID, maskResourceID string) (*ebiten.Image, error) {
	// Get file paths from resource config
	basePath, exists := rm.resourceMap[baseResourceID]
	if !exists {
		return nil, fmt.Errorf("base resource ID not found: %s", baseResourceID)
	}
	maskPath, exists := rm.resourceMap[maskResourceID]
	if !exists {
		return nil, fmt.Errorf("mask resource ID not found: %s", maskResourceID)
	}

	// resourceMap already contains the full path with BasePath prepended
	// (constructed by buildResourceMap), so we can use it directly

	// Load base image file from embedded FS
	baseFile, err := embedded.Open(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open base image file %s: %w", basePath, err)
	}
	defer baseFile.Close()

	baseImg, _, err := image.Decode(baseFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base image %s: %w", basePath, err)
	}

	// Load mask image file from embedded FS
	maskFile, err := embedded.Open(maskPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open mask image file %s: %w", maskPath, err)
	}
	defer maskFile.Close()

	maskImg, format, err := image.Decode(maskFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode mask image %s: %w", maskPath, err)
	}

	log.Printf("[ResourceManager] Applying alpha mask: %s (format=%s)", maskResourceID, format)

	// Apply alpha mask
	bounds := baseImg.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create RGBA image for result
	result := image.NewRGBA(bounds)

	// Process each pixel
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get base pixel
			br, bg, bb, _ := baseImg.At(x, y).RGBA()

			// Get mask pixel and convert to alpha
			mr, mg, mb, ma := maskImg.At(x, y).RGBA()

			// Calculate alpha from mask
			var alpha uint8
			if ma < 65535 {
				// PNG has alpha channel, use it directly
				alpha = uint8(ma / 256)
			} else {
				// PNG is RGB only, use brightness as alpha
				// White pixels in mask = fully opaque (keep the JPG pixel)
				// Black pixels in mask = fully transparent (remove background)
				// Gray pixels = partial transparency (anti-aliasing)
				maskBrightness := (mr + mg + mb) / 3
				alpha = uint8(maskBrightness / 256)
			}

			// Apply premultiplied alpha to reduce edge artifacts
			// This ensures semi-transparent edges blend correctly
			alphaF := float64(alpha) / 255.0
			finalR := uint8(float64(br/256) * alphaF)
			finalG := uint8(float64(bg/256) * alphaF)
			finalB := uint8(float64(bb/256) * alphaF)

			// Set result pixel
			result.Set(x, y, color.RGBA{
				R: finalR,
				G: finalG,
				B: finalB,
				A: alpha,
			})
		}
	}

	// Convert to ebiten.Image
	return ebiten.NewImageFromImage(result), nil
}

// GetImageMetadata retrieves sprite sheet metadata (cols, rows) for an image resource.
// Returns (cols, rows, true) if the resource exists and has metadata, or (0, 0, false) otherwise.
//
// Parameters:
//   - resourceID: The resource ID (e.g., "IMAGE_DIRTSMALL")
//
// Returns:
//   - cols: Number of columns in the sprite sheet (0 if not a sprite sheet)
//   - rows: Number of rows in the sprite sheet (0 if not a sprite sheet)
//   - ok: true if the resource was found, false otherwise
//
// Example:
//
//	cols, rows, ok := rm.GetImageMetadata("IMAGE_DIRTSMALL")
//	if ok {
//	    log.Printf("Sprite sheet: %d cols × %d rows", cols, rows)
//	}
func (rm *ResourceManager) GetImageMetadata(resourceID string) (cols int, rows int, ok bool) {
	if rm.config == nil {
		return 0, 0, false
	}

	// Search for the image resource in all groups
	for _, group := range rm.config.Groups {
		for _, img := range group.Images {
			if img.ID == resourceID {
				return img.Cols, img.Rows, true
			}
		}
	}

	return 0, 0, false
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
			// 记录警告但继续加载其他资源（粒子查看器模式）
			log.Printf("Warning: Failed to load image %s in group %s: %v (skipping)", img.ID, groupName, err)
			continue
		}
	}

	// Load all sounds in the group
	for _, sound := range group.Sounds {
		// Look up the file path
		filePath, exists := rm.resourceMap[sound.ID]
		if !exists {
			log.Printf("Warning: Sound resource ID not found: %s (skipping)", sound.ID)
			continue
		}

		// Load the sound effect (use LoadSoundEffect for non-looping sounds)
		if _, err := rm.LoadSoundEffect(filePath); err != nil {
			log.Printf("Warning: Failed to load sound %s in group %s: %v (skipping)", sound.ID, groupName, err)
			continue
		}
	}

	// Fonts are not loaded here as they require a size parameter
	// They should be loaded individually using LoadFont when needed

	return nil
}

// LoadAllResources loads all resource groups defined in the configuration file.
// This is a convenience method to batch-load all game resources at once,
// typically used during game initialization or for verification/testing programs.
//
// The function iterates through all resource groups in the YAML configuration
// and loads each one. If any group fails to load, it logs a warning and continues
// loading other groups (fail-soft behavior).
//
// Returns:
//   - An error only if the resource config hasn't been loaded yet
//   - Individual resource loading errors are logged but don't stop the process
//
// Example:
//
//	// Load all resources at startup
//	if err := rm.LoadAllResources(); err != nil {
//	    log.Fatal("Failed to start resource loading:", err)
//	}
func (rm *ResourceManager) LoadAllResources() error {
	// Check if resource config is loaded
	if rm.config == nil {
		return fmt.Errorf("resource config not loaded - call LoadResourceConfig first")
	}

	log.Printf("[ResourceManager] Loading all resource groups (%d groups total)...", len(rm.config.Groups))

	// Load each resource group
	loadedCount := 0
	for groupName := range rm.config.Groups {
		if err := rm.LoadResourceGroup(groupName); err != nil {
			log.Printf("Warning: Failed to load resource group %s: %v (continuing)", groupName, err)
			continue
		}
		loadedCount++
	}

	log.Printf("[ResourceManager] Successfully loaded %d/%d resource groups", loadedCount, len(rm.config.Groups))
	return nil
}

// LoadParticleConfig loads a particle configuration file from the particles directory and caches it.
// If the configuration has already been loaded, it returns the cached version.
//
// Parameters:
//   - name: The particle configuration name. Can be either:
//   - A simple name (e.g., "Award", "BossExplosion") - will be loaded from data/particles/
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
		path = filepath.Join("data/particles", name+".xml")
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

// LoadBitmapFont loads a bitmap font from PNG image and TXT metadata files
// Story 8.2: 用于加载原版 PVZ 的位图字体（如 HouseofTerror28）
//
// Parameters:
//   - imagePath: PNG 图集文件路径（如 "assets/data/HouseofTerror28.png"）
//   - metaPath: TXT 元数据文件路径（如 "assets/data/HouseofTerror28.txt"）
//
// Returns:
//   - *utils.BitmapFont: 加载的位图字体实例
//   - error: 如果加载失败
//
// Example usage:
//
//	font, err := rm.LoadBitmapFont(
//	    "assets/data/HouseofTerror28.png",
//	    "assets/data/HouseofTerror28.txt",
//	)
//	if err != nil {
//	    log.Printf("Failed to load bitmap font: %v", err)
//	}
func (rm *ResourceManager) LoadBitmapFont(imagePath, metaPath string) (*utils.BitmapFont, error) {
	// 使用 imagePath 作为缓存键
	if cached, ok := rm.bitmapFontCache[imagePath]; ok {
		return cached, nil
	}

	// 加载位图字体
	font, err := utils.LoadBitmapFont(imagePath, metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load bitmap font from %s: %w", imagePath, err)
	}

	// 缓存字体
	rm.bitmapFontCache[imagePath] = font
	log.Printf("Loaded bitmap font: %s", imagePath)

	return font, nil
}

// GetBitmapFont retrieves a cached bitmap font by image path.
// Returns nil if the font has not been loaded yet.
//
// Parameters:
//   - imagePath: PNG 图集文件路径
//
// Returns:
//   - *utils.BitmapFont: 缓存的字体实例，如果未加载则返回 nil
//
// Example usage:
//
//	font := rm.GetBitmapFont("assets/data/HouseofTerror28.png")
//	if font == nil {
//	    log.Println("Font not loaded yet")
//	}
func (rm *ResourceManager) GetBitmapFont(imagePath string) *utils.BitmapFont {
	return rm.bitmapFontCache[imagePath]
}

// ========================================
// Story 13.6: Reanim 配置管理器方法
// ========================================

// SetReanimConfigManager 设置 Reanim 配置管理器
//
// 此方法由游戏初始化逻辑调用，设置全局配置管理器。
// 配置管理器将被传递给 ReanimSystem，用于配置驱动的动画播放。
//
// Parameters:
//   - manager: 配置管理器实例
//
// Example usage:
//
//	reanimConfigManager, err := config.NewReanimConfigManager("data/reanim_config.yaml")
//	if err != nil {
//	    log.Fatalf("Failed to load config: %v", err)
//	}
//	resourceManager.SetReanimConfigManager(reanimConfigManager)
func (rm *ResourceManager) SetReanimConfigManager(manager *config.ReanimConfigManager) {
	rm.reanimConfigManager = manager
	log.Printf("[ResourceManager] Reanim 配置管理器已设置")
}

// GetReanimConfigManager 获取 Reanim 配置管理器
//
// Returns:
//   - *config.ReanimConfigManager: 配置管理器实例，如果未设置则返回 nil
//
// Example usage:
//
//	configManager := resourceManager.GetReanimConfigManager()
//	if configManager != nil {
//	    // 使用配置管理器
//	}
func (rm *ResourceManager) GetReanimConfigManager() *config.ReanimConfigManager {
	return rm.reanimConfigManager
}

// ===== Story 8.8: 背景音乐淡出功能 =====

// SetCurrentBGM 设置当前播放的背景音乐播放器
// 用于在淡出时能够找到并控制背景音乐
//
// Parameters:
//   - player: 当前播放的背景音乐播放器
//
// Example usage:
//
//	player, err := rm.LoadAudio("assets/audio/Music/mainmenu.ogg")
//	if err == nil {
//	    rm.SetCurrentBGM(player)
//	    player.Play()
//	}
func (rm *ResourceManager) SetCurrentBGM(player *audio.Player) {
	rm.currentBGMPlayer = player
}

// FadeOutMusic 开始淡出当前背景音乐
// 在指定的时间内将音量从当前值降至 0，然后停止播放
//
// 注意：
//   - 需要在每帧调用 UpdateBGMFade(deltaTime) 来更新淡出进度
//   - 如果没有设置当前背景音乐播放器，此方法不执行任何操作
//
// Parameters:
//   - duration: 淡出时长（秒），例如 0.2 表示 200 毫秒
//
// Example usage:
//
//	// 在游戏冻结时淡出背景音乐
//	rm.FadeOutMusic(0.2)
func (rm *ResourceManager) FadeOutMusic(duration float64) {
	if rm.currentBGMPlayer == nil {
		return // 没有背景音乐播放器，跳过
	}

	// 记录当前音量作为初始音量
	rm.bgmInitialVolume = rm.currentBGMPlayer.Volume()
	rm.bgmFadeDuration = duration
	rm.bgmFadeElapsed = 0.0
	rm.bgmFadeOut = true

	log.Printf("[ResourceManager] 开始淡出背景音乐，时长: %.2f 秒，初始音量: %.2f", duration, rm.bgmInitialVolume)
}

// UpdateBGMFade 更新背景音乐淡出进度
// 应在每帧调用（通常在 GameScene.Update() 中）
//
// Parameters:
//   - deltaTime: 自上次更新以来的时间（秒）
//
// Example usage:
//
//	// 在 GameScene.Update() 中
//	rm.UpdateBGMFade(deltaTime)
func (rm *ResourceManager) UpdateBGMFade(deltaTime float64) {
	if !rm.bgmFadeOut || rm.currentBGMPlayer == nil {
		return // 没有正在进行的淡出
	}

	// 更新淡出进度
	rm.bgmFadeElapsed += deltaTime

	// 计算当前音量（线性插值）
	progress := rm.bgmFadeElapsed / rm.bgmFadeDuration
	if progress >= 1.0 {
		// 淡出完成
		rm.currentBGMPlayer.SetVolume(0.0)
		rm.currentBGMPlayer.Pause() // 停止播放
		rm.bgmFadeOut = false
		log.Printf("[ResourceManager] 背景音乐淡出完成")
	} else {
		// 线性淡出：从初始音量降到 0
		currentVolume := rm.bgmInitialVolume * (1.0 - progress)
		rm.currentBGMPlayer.SetVolume(currentVolume)
	}
}

// IsBGMFadingOut 检查背景音乐是否正在淡出
//
// Returns:
//   - bool: 如果正在淡出返回 true，否则返回 false
func (rm *ResourceManager) IsBGMFadingOut() bool {
	return rm.bgmFadeOut
}
