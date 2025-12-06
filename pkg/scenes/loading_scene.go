package scenes

import (
	"image"
	"image/color"
	"log"
	"math"
	"strings"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// LoadingScene represents the loading screen shown when the game starts.
// It displays a progress bar, logo animation, and loading messages.
type LoadingScene struct {
	resourceManager *game.ResourceManager
	sceneManager    *game.SceneManager

	// Progress tracking
	progress         float64 // Loading progress (0.0 - 1.0)
	loadingComplete  bool    // Whether loading is complete
	elapsedTime      float64 // Elapsed time since scene start
	logoY            float64 // Current Y position of the logo (for drop animation)
	logoAnimComplete bool    // Whether logo animation has completed

	// Image resources
	backgroundImage *ebiten.Image // Title screen background
	logoImage       *ebiten.Image // PvZ logo (RGB + Alpha composited)
	logoRGB         *ebiten.Image // Logo RGB base image (for delayed composition)
	logoMask        *ebiten.Image // Logo Alpha mask (for delayed composition)
	dirtBarImage    *ebiten.Image // Progress bar dirt base
	grassBarImage   *ebiten.Image // Progress bar grass fill
	logoComposited  bool          // Whether logo has been composited

	// Font resources
	textFontFace *text.GoTextFace // Font for loading messages

	// Audio players (not used for now - will implement in Task 6)
	// flowerPlayer *audio.Player
	// zombiePlayer *audio.Player
	// clickPlayer  *audio.Player

	// Sod roll cap (simple linear interpolation, no ECS)
	sodRollCapImage *ebiten.Image // Sod roll cap image

	// ECS system for Reanim animations (only for sprouts)
	entityManager   *ecs.EntityManager
	reanimSystem    *systems.ReanimSystem
	renderSystem    *systems.RenderSystem
	configManager   *config.ReanimConfigManager
	cameraX         float64                        // Camera X position (0 for loading scene)
	cameraY         float64                        // Camera Y position (0 for loading scene)
	sproutEntities  []ecs.EntityID                 // Sprout animation entities
	triggeredFlags  []bool                         // Track which sprout animations have been triggered
	sproutPositions map[int]struct{ x, y float64 } // Position for each sprout animation

	// Mouse interaction state
	isHoveringProgressBar bool // Whether mouse is hovering over progress bar

	// Sound timing
	lastSproutSoundTime float64  // Last time a sprout/zombie sound was played
	pendingSounds       []string // Queue of pending sounds to play
}

// NewLoadingScene creates a new loading scene.
func NewLoadingScene(rm *game.ResourceManager, sm *game.SceneManager, configManager *config.ReanimConfigManager) *LoadingScene {
	scene := &LoadingScene{
		resourceManager: rm,
		sceneManager:    sm,
		configManager:   configManager,
		progress:        0.0,
		loadingComplete: false,
		elapsedTime:     0.0,
		logoY:           -200, // Start above screen
		cameraX:         0,
		cameraY:         0,
		triggeredFlags:  make([]bool, len(config.LoadingSproutTriggers)),
		sproutPositions: make(map[int]struct{ x, y float64 }),
	}

	// Load background image
	var err error
	scene.backgroundImage, err = rm.LoadImage("assets/images/titlescreen.jpg")
	if err != nil {
		log.Printf("Failed to load background image: %v", err)
	}

	// Load and composite logo (RGB + Alpha mask)
	// Note: Composition is delayed until first Draw() to avoid ReadPixels panic
	// Load RGB base image
	logoRGB, err := rm.LoadImage("assets/images/PvZ_Logo.jpg")
	if err != nil {
		log.Printf("Failed to load logo RGB image: %v", err)
	} else {
		scene.logoRGB = logoRGB
	}

	// Load Alpha mask image
	logoMask, err := rm.LoadImage("assets/images/PvZ_Logo_.png")
	if err != nil {
		log.Printf("Failed to load logo mask image: %v", err)
	} else {
		scene.logoMask = logoMask
	}

	// Mark logo as not composited yet (will be done in first Draw())
	scene.logoComposited = false

	// Load progress bar images
	scene.dirtBarImage, err = rm.LoadImage("assets/images/LoadBar_dirt.png")
	if err != nil {
		log.Printf("Failed to load dirt bar image: %v", err)
	}

	scene.grassBarImage, err = rm.LoadImage("assets/images/LoadBar_grass.png")
	if err != nil {
		log.Printf("Failed to load grass bar image: %v", err)
	}

	// Load sod roll cap image (single image, will apply transform)
	scene.sodRollCapImage, err = rm.LoadImage("assets/reanim/SodRollCap.png")
	if err != nil {
		log.Printf("[LoadingScene] Failed to load sod roll cap image: %v", err)
	}

	// Load fonts
	scene.loadFonts()

	// Initialize ECS systems (only for sprouts)
	scene.initECS()

	// Define sprout positions (spread across progress bar)
	barWidth := 314.0 // Grass bar width
	for i := range config.LoadingSproutTriggers {
		// Position sprouts above and slightly offset from progress bar
		baseX := config.LoadingBarX + config.LoadingGrassOffsetX
		offsetX := barWidth * float64(config.LoadingSproutTriggers[i])

		// Get Y offset for this specific sprout (or use default if not configured)
		offsetY := -40.0 // Default offset
		if i < len(config.LoadingSproutOffsetsY) {
			offsetY = config.LoadingSproutOffsetsY[i]
		}

		scene.sproutPositions[i] = struct{ x, y float64 }{
			x: baseX + offsetX,
			y: config.LoadingBarY + offsetY,
		}
	}

	// 播放背景音乐
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlayMusic("SOUND_MAINMUSIC")
	}

	return scene
}

// getUnitIDFromResourceName converts ResourceManager name (PascalCase) to config unitID (lowercase)
// Examples: "LoadBar_sprout" -> "loadbar_sprout", "LoadBar_Zombiehead" -> "loadbar_zombiehead"
func getUnitIDFromResourceName(resourceName string) string {
	// Simple conversion: just lowercase the entire name
	return strings.ToLower(resourceName)
}

// loadFonts loads the font resources for the loading scene.
func (s *LoadingScene) loadFonts() {
	// Load text font for loading messages
	var err error
	s.textFontFace, err = s.resourceManager.LoadFont("assets/fonts/SimHei.ttf", config.LoadingTextFontSize)
	if err != nil {
		log.Printf("Failed to load text font: %v", err)
	}
}

// initECS initializes the ECS systems for Reanim animations (sprouts only).
func (s *LoadingScene) initECS() {
	s.entityManager = ecs.NewEntityManager()
	s.reanimSystem = systems.NewReanimSystem(s.entityManager)
	s.reanimSystem.SetConfigManager(s.configManager)
	s.renderSystem = systems.NewRenderSystem(s.entityManager)
	s.renderSystem.SetReanimSystem(s.reanimSystem) // 设置 ReanimSystem 引用，用于 GetRenderData
}

// Update updates the loading scene logic.
func (s *LoadingScene) Update(deltaTime float64) {
	s.elapsedTime += deltaTime

	// Update logo drop animation
	s.updateLogoAnimation(deltaTime)

	// Update loading progress
	s.updateProgress(deltaTime)

	// Check and trigger sprout animations
	s.checkSproutTriggers()

	// Process pending sounds with minimum interval
	s.processPendingSounds()

	// Update ECS systems (for sprouts)
	s.reanimSystem.Update(deltaTime)

	// Check for click to transition to main menu (only when complete and hovering over progress bar)
	if s.loadingComplete {
		s.updateMouseInteraction()
	}
}

// processPendingSounds plays pending sounds with minimum interval between them.
func (s *LoadingScene) processPendingSounds() {
	if len(s.pendingSounds) == 0 {
		return
	}

	// Check if enough time has passed since last sound
	if s.elapsedTime-s.lastSproutSoundTime >= config.LoadingSoundMinInterval {
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			// Play the first sound in queue
			audioManager.PlaySound(s.pendingSounds[0])
			s.lastSproutSoundTime = s.elapsedTime
			// Remove from queue
			s.pendingSounds = s.pendingSounds[1:]
		}
	}
}

// updateLogoAnimation updates the logo drop animation.
func (s *LoadingScene) updateLogoAnimation(deltaTime float64) {
	if s.logoAnimComplete {
		return
	}

	if s.elapsedTime < config.LoadingLogoAnimDuration {
		// Calculate animation progress (0.0 - 1.0)
		t := s.elapsedTime / config.LoadingLogoAnimDuration

		// Apply EaseOutQuad easing
		t = t * (2 - t)

		// Calculate logo Y position
		startY := -200.0 // Start above screen
		s.logoY = startY + (config.LoadingLogoTargetY-startY)*t
	} else {
		s.logoY = config.LoadingLogoTargetY
		s.logoAnimComplete = true
	}
}

// updateProgress updates the loading progress.
func (s *LoadingScene) updateProgress(deltaTime float64) {
	if s.loadingComplete {
		return
	}

	// Simulate loading progress (uniform speed)
	s.progress += deltaTime / config.LoadingDuration
	s.progress = math.Min(s.progress, 1.0)

	// Check if loading is complete
	if s.progress >= 1.0 && !s.loadingComplete {
		s.loadingComplete = true
		// TODO: Play flower sound (Task 6)
	}
}

// updateMouseInteraction handles mouse hover and click interaction with progress bar.
func (s *LoadingScene) updateMouseInteraction() {
	// Get mouse position
	mouseX, mouseY := ebiten.CursorPosition()

	// Calculate progress bar bounds
	// Dirt bar position and size
	dirtBarX := config.LoadingBarX
	dirtBarY := config.LoadingBarY
	dirtBarWidth := 321.0 // LoadBar_dirt.png width
	dirtBarHeight := 53.0 // LoadBar_dirt.png height

	// Check if mouse is hovering over progress bar
	s.isHoveringProgressBar = float64(mouseX) >= dirtBarX &&
		float64(mouseX) <= dirtBarX+dirtBarWidth &&
		float64(mouseY) >= dirtBarY &&
		float64(mouseY) <= dirtBarY+dirtBarHeight

	// Set cursor type based on hover state
	if s.isHoveringProgressBar {
		ebiten.SetCursorShape(ebiten.CursorShapePointer) // Hand cursor
	} else {
		ebiten.SetCursorShape(ebiten.CursorShapeDefault) // Normal cursor
	}

	// Check for click (only when hovering)
	if s.isHoveringProgressBar && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		s.onClickStart()
	}
}

// checkSproutTriggers checks and triggers sprout animations at progress thresholds.
func (s *LoadingScene) checkSproutTriggers() {
	for i, threshold := range config.LoadingSproutTriggers {
		if !s.triggeredFlags[i] && s.progress >= threshold {
			s.triggeredFlags[i] = true
			s.spawnSproutAnimation(i)
		}
	}
}

// spawnSproutAnimation creates a sprout animation entity.
func (s *LoadingScene) spawnSproutAnimation(index int) {
	entity := s.entityManager.CreateEntity()

	// Get position for this sprout
	pos := s.sproutPositions[index]

	ecs.AddComponent(s.entityManager, entity, &components.PositionComponent{
		X: pos.x,
		Y: pos.y + 10,
	})

	// Determine animation type based on index
	var resourceName string  // ResourceManager uses PascalCase names
	var animationName string // Direct animation name to play
	var soundID string       // Sound to play

	if index == len(config.LoadingSproutTriggers)-1 {
		// Last trigger: zombie head
		resourceName = "LoadBar_Zombiehead"
		animationName = "anim_zombie" // From loadbar_zombiehead.yaml default_animation
		soundID = "SOUND_LOADINGBAR_ZOMBIE"
	} else {
		// Other triggers: sprout with variations
		resourceName = "LoadBar_sprout"
		animationName = "anim_sprout" // From loadbar_sprout.yaml default_animation
		soundID = "SOUND_LOADINGBAR_FLOWER"
	}

	// 将音效加入待播放队列
	s.pendingSounds = append(s.pendingSounds, soundID)

	// Load Reanim data from ResourceManager
	reanimXML := s.resourceManager.GetReanimXML(resourceName)
	if reanimXML == nil {
		log.Printf("[LoadingScene] Failed to load %s reanim (not found in cache)", resourceName)
		return
	}

	partImages := s.resourceManager.GetReanimPartImages(resourceName)
	if partImages == nil {
		log.Printf("[LoadingScene] Failed to load %s part images (not found in cache)", resourceName)
		return
	}

	// Add ReanimComponent with full data
	ecs.AddComponent(s.entityManager, entity, &components.ReanimComponent{
		ReanimName: resourceName,
		ReanimXML:  reanimXML,
		PartImages: partImages,
	})

	// Add animation command
	// Note: Need to set UnitID so ReanimSystem can read loop config from YAML
	ecs.AddComponent(s.entityManager, entity, &components.AnimationCommandComponent{
		UnitID:        getUnitIDFromResourceName(resourceName), // Convert ResourceName to unitID
		AnimationName: animationName,
		Processed:     false,
	})

	// Apply scale variations (Story 1.5 Task 4)
	// - 0.2 (index 0): Normal (ScaleX=1.0, ScaleY=1.0)
	// - 0.4 (index 1): Mirror (ScaleX=-1.0, ScaleY=1.0)
	// - 0.6 (index 2): Enlarged (ScaleX=1.5, ScaleY=1.5)
	// - 0.8 (index 3): Mirror (ScaleX=-1.0, ScaleY=1.0)
	// - 1.0 (index 4): Zombie head (Normal)
	var scaleX, scaleY float64
	if index == len(config.LoadingSproutTriggers)-1 {
		// Zombie head: Normal
		scaleX, scaleY = 1.0, 1.0
	} else {
		switch index {
		case 0: // 0.2: Normal
			scaleX, scaleY = 1.0, 1.0
		case 1: // 0.4: Mirror
			scaleX, scaleY = -1.0, 1.0
		case 2: // 0.6: Enlarged
			scaleX, scaleY = 1.5, 1.5
		case 3: // 0.8: Mirror
			scaleX, scaleY = -1.0, 1.0
		default:
			scaleX, scaleY = 1.0, 1.0
		}
	}
	ecs.AddComponent(s.entityManager, entity, &components.ScaleComponent{
		ScaleX: scaleX,
		ScaleY: scaleY,
	})

	// Store entity for cleanup
	s.sproutEntities = append(s.sproutEntities, entity)
}

// onClickStart handles the click event when loading is complete.
func (s *LoadingScene) onClickStart() {
	// 播放开始按钮点击音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_BUTTONCLICK")
	}

	// Transition to main menu scene
	mainMenuScene := NewMainMenuScene(s.resourceManager, s.sceneManager)
	s.sceneManager.SwitchTo(mainMenuScene)
}

// Draw renders the loading scene to the screen.
func (s *LoadingScene) Draw(screen *ebiten.Image) {
	// Apply alpha mask composition on first draw
	// (must be done after game loop starts to avoid ReadPixels panic)
	if !s.logoComposited && s.logoRGB != nil && s.logoMask != nil {
		s.logoImage = utils.ApplyAlphaMask(s.logoRGB, s.logoMask)
		s.logoComposited = true
		log.Printf("[LoadingScene] Logo alpha mask applied")
	}

	// Draw background
	s.drawBackground(screen)

	// Draw logo
	s.drawLogo(screen)

	// Draw progress bar (includes sod roll cap)
	s.drawProgressBar(screen)

	// Draw Reanim animations (sprouts only)
	s.renderSystem.DrawGameWorld(screen, s.cameraX)

	// Draw text messages
	s.drawText(screen)
}

// drawBackground draws the background image.
func (s *LoadingScene) drawBackground(screen *ebiten.Image) {
	if s.backgroundImage == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	screen.DrawImage(s.backgroundImage, op)
}

// drawLogo draws the PvZ logo.
func (s *LoadingScene) drawLogo(screen *ebiten.Image) {
	if s.logoImage == nil {
		return
	}

	// Draw logo centered horizontally
	logoBounds := s.logoImage.Bounds()
	logoWidth := float64(logoBounds.Dx())
	logoX := (WindowWidth - logoWidth) / 2.0

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(logoX, s.logoY)
	screen.DrawImage(s.logoImage, op)
}

// drawProgressBar draws the progress bar (dirt base and grass fill).
func (s *LoadingScene) drawProgressBar(screen *ebiten.Image) {
	// Draw dirt bar base
	if s.dirtBarImage != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(config.LoadingBarX, config.LoadingBarY)
		screen.DrawImage(s.dirtBarImage, op)
	}

	// Draw grass bar (cropped by progress)
	if s.grassBarImage != nil {
		grassBounds := s.grassBarImage.Bounds()
		grassWidth := grassBounds.Dx()
		grassHeight := grassBounds.Dy()

		// Calculate visible width based on progress
		visibleWidth := int(float64(grassWidth) * s.progress)

		if visibleWidth > 0 {
			// Crop grass image
			visibleGrass := s.grassBarImage.SubImage(
				image.Rect(0, 0, visibleWidth, grassHeight),
			).(*ebiten.Image)

			// Draw cropped grass
			op := &ebiten.DrawImageOptions{}
			grassX := config.LoadingBarX + config.LoadingGrassOffsetX
			grassY := config.LoadingBarY + config.LoadingGrassOffsetY
			op.GeoM.Translate(grassX, grassY)
			screen.DrawImage(visibleGrass, op)
		}
	}

	// Draw sod roll cap (on top of grass)
	s.drawSodRollCap(screen)
}

// drawText draws the loading text messages.
func (s *LoadingScene) drawText(screen *ebiten.Image) {
	if s.textFontFace == nil {
		return
	}

	var message string
	var textColor color.Color

	if s.loadingComplete {
		message = "点击开始"
		// 如果鼠标悬停在进度条上，使用土黄色（与"载入中……"相同）
		// 否则使用红色
		if s.isHoveringProgressBar {
			textColor = color.RGBA{218, 165, 32, 255} // Goldenrod (same as loading)
		} else {
			textColor = color.RGBA{236, 78, 32, 255} // Red
		}
	} else {
		message = "载入中……"
		textColor = color.RGBA{218, 165, 32, 255} // Goldenrod
	}

	// Calculate text position (centered)
	textWidth := float64(len(message)) * config.LoadingTextFontSize * 0.5 // Approximate width
	textX := (WindowWidth-textWidth)/2.0 + config.LoadingTextOffsetX
	textY := config.LoadingTextY

	// Draw shadow (black, offset)
	shadowOp := &text.DrawOptions{}
	shadowOp.GeoM.Translate(textX+2, textY+2)
	shadowOp.ColorScale.ScaleWithColor(color.Black)
	text.Draw(screen, message, s.textFontFace, shadowOp)

	// Draw main text
	mainOp := &text.DrawOptions{}
	mainOp.GeoM.Translate(textX, textY)
	mainOp.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, message, s.textFontFace, mainOp)
}

// drawSodRollCap draws the sod roll cap with linear interpolation between keyframes.
func (s *LoadingScene) drawSodRollCap(screen *ebiten.Image) {
	if s.sodRollCapImage == nil || s.progress <= 0 || s.progress >= 1.0 {
		return
	}

	// Keyframe data from SodRoll.reanim
	// 帧 0:  ScaleX=0.8,   ScaleY=0.8
	// 帧 13: ScaleX=0.624, ScaleY=0.624
	const (
		frame0Scale  = 0.8
		frame13Scale = 0.624
	)

	// Linear interpolation for scale based on progress (0.0 → 1.0)
	currentScale := frame0Scale + (frame13Scale-frame0Scale)*s.progress

	// Calculate position
	grassWidth := 314.0 // Grass bar width
	grassX := config.LoadingBarX + config.LoadingGrassOffsetX
	grassY := config.LoadingBarY + config.LoadingGrassOffsetY

	// X: follow grass bar right edge
	// Cap should be centered at the grass bar right edge
	imgBounds := s.sodRollCapImage.Bounds()
	imgWidth := float64(imgBounds.Dx())
	imgHeight := float64(imgBounds.Dy())
	scaledWidth := imgWidth * currentScale

	grassRightEdge := grassX + (grassWidth * s.progress)
	capX := grassRightEdge - (scaledWidth / 2.0) // Center the cap at grass right edge

	// Y: place cap on top of grass bar
	// Cap image height is 71, we want the bottom of the scaled image to touch grassY
	scaledHeight := imgHeight * currentScale

	// Position so bottom edge is slightly below grassY for visual grounding
	// Adjustable offset to fine-tune visual contact
	const yOffset = 30.0 // Positive = move down
	capY := grassY - scaledHeight + yOffset

	// Calculate rotation for rolling effect
	// As the cap moves from left to right, it should rotate
	// Full progress (314 pixels) = multiple full rotations
	// Assuming cap diameter ~73 pixels, circumference = π * 73 ≈ 229
	// So 314 pixels ≈ 1.37 rotations ≈ 1.37 * 2π radians
	const rotationsPerFullProgress = 1.5 // Adjust for visual effect
	rotation := s.progress * rotationsPerFullProgress * 2 * math.Pi

	// Draw with scale and rotation
	op := &ebiten.DrawImageOptions{}
	// 1. Move image center to origin (for rotation around center)
	op.GeoM.Translate(-imgWidth/2.0, -imgHeight/2.0)
	// 2. Apply rotation (rolling effect)
	op.GeoM.Rotate(rotation)
	// 3. Apply scale
	op.GeoM.Scale(currentScale, currentScale)
	// 4. Move to final position (adjust for the fact we rotated around center)
	op.GeoM.Translate(capX+scaledWidth/2.0, capY+scaledHeight/2.0)

	screen.DrawImage(s.sodRollCapImage, op)
}
