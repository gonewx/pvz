package scenes

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/hajimehoshi/ebiten/v2"
)

// loadResources loads all UI images required for the game scene.
// If a resource fails to load, it logs a warning but continues.
// The Draw method will use fallback rendering for missing resources.
func (s *GameScene) loadResources() {
	// Story 8.2 QA改进：根据关卡配置加载背景
	// 如果关卡配置了特定背景，使用配置的背景；否则使用默认背景
	backgroundImageID := "IMAGE_BACKGROUND1"
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.BackgroundImage != "" {
		backgroundImageID = s.gameState.CurrentLevel.BackgroundImage
		log.Printf("[GameScene] 使用关卡配置的背景: %s", backgroundImageID)
	}

	// Load lawn background
	// CRITICAL: 创建背景副本，避免修改缓存中的原始图片
	// 问题说明：ResourceManager 会缓存加载的图片，如果直接在缓存图片上预渲染草皮，
	// 当从 Level 1-4 过渡到 Level 1-5 时，IMAGE_BACKGROUND1 仍然包含 1-4 预渲染的草皮
	// 解决方案：每个场景都使用独立的背景副本
	cachedBg, err := s.resourceManager.LoadImageByID(backgroundImageID)
	if err != nil {
		log.Printf("Warning: Failed to load lawn background %s: %v", backgroundImageID, err)
		log.Printf("Will use fallback solid color background")
	} else {
		// 创建背景副本
		bounds := cachedBg.Bounds()
		bgCopy := ebiten.NewImage(bounds.Dx(), bounds.Dy())
		op := &ebiten.DrawImageOptions{}
		bgCopy.DrawImage(cachedBg, op)
		s.background = bgCopy
		log.Printf("[GameScene] 创建背景副本: %s (%dx%d)", backgroundImageID, bounds.Dx(), bounds.Dy())

		// Calculate maximum camera position (rightmost edge)
		bgWidth := bounds.Dx()
		s.maxCameraX = float64(bgWidth - WindowWidth)
		if s.maxCameraX < 0 {
			s.maxCameraX = 0 // Background is smaller than window
		}
	}

	// Load seed bank (植物选择栏背景)
	seedBank, err := s.resourceManager.LoadImageByID("IMAGE_SEEDBANK")
	if err != nil {
		log.Printf("Warning: Failed to load seed bank image: %v", err)
		log.Printf("Will use fallback rendering for seed bank")
	} else {
		s.seedBank = seedBank
	}

	// Load shovel slot background
	shovelSlot, err := s.resourceManager.LoadImageByID("IMAGE_SHOVELBANK")
	if err != nil {
		log.Printf("Warning: Failed to load shovel slot: %v", err)
	} else {
		s.shovelSlot = shovelSlot
	}

	// Load shovel icon
	shovel, err := s.resourceManager.LoadImageByID("IMAGE_SHOVEL")
	if err != nil {
		log.Printf("Warning: Failed to load shovel icon: %v", err)
	} else {
		s.shovel = shovel
	}

	// 铲子槽点击音效由 AudioManager 统一管理（Story 10.9）

	// Load font for sun counter (使用黑体)
	font, err := s.resourceManager.LoadFont("assets/fonts/SimHei.ttf", config.SunCounterFontSize)
	if err != nil {
		log.Printf("Warning: Failed to load sun counter font: %v", err)
		log.Printf("Will use fallback debug text rendering")
	} else {
		s.sunCounterFont = font
	}

	// Load font for plant card sun cost (使用黑体，字体大小从配置读取)
	cardFont, err := s.resourceManager.LoadFont("assets/fonts/SimHei.ttf", float64(config.PlantCardSunCostFontSize))
	if err != nil {
		log.Printf("Warning: Failed to load plant card font: %v", err)
		log.Printf("Will use fallback debug text rendering for card cost")
	} else {
		s.plantCardFont = cardFont
	}

	// Load progress bar resources (进度条资源)
	flagMeter, err := s.resourceManager.LoadImageByID("IMAGE_FLAGMETER")
	if err != nil {
		log.Printf("Warning: Failed to load progress bar background: %v", err)
	} else {
		s.flagMeter = flagMeter
	}

	flagMeterProg, err := s.resourceManager.LoadImageByID("IMAGE_FLAGMETERLEVELPROGRESS")
	if err != nil {
		log.Printf("Warning: Failed to load progress bar fill: %v", err)
	} else {
		s.flagMeterProg = flagMeterProg
	}

	flagMeterFlag, err := s.resourceManager.LoadImageByID("IMAGE_FLAGMETERPARTS")
	if err != nil {
		log.Printf("Warning: Failed to load progress bar flags: %v", err)
	} else {
		s.flagMeterFlag = flagMeterFlag
	}

	// Story 19.4: Load bowling red line image (保龄球红线图片)
	bowlingRedLine, err := s.resourceManager.LoadImage("assets/images/Wallnut_bowlingstripe.png")
	if err != nil {
		log.Printf("Warning: Failed to load bowling red line image: %v", err)
	} else {
		s.bowlingRedLine = bowlingRedLine
		log.Printf("[GameScene] Loaded bowling red line image")
	}

	// Story 19.5: Load conveyor belt images (传送带图片)
	conveyorBackdrop, err := s.resourceManager.LoadImageByID("IMAGE_CONVEYORBELT_BACKDROP")
	if err != nil {
		log.Printf("Warning: Failed to load conveyor belt backdrop: %v", err)
	} else {
		s.conveyorBeltBackdrop = conveyorBackdrop
		log.Printf("[GameScene] Loaded conveyor belt backdrop")
	}

	conveyorBelt, err := s.resourceManager.LoadImageByID("IMAGE_CONVEYORBELT")
	if err != nil {
		log.Printf("Warning: Failed to load conveyor belt animation: %v", err)
	} else {
		s.conveyorBelt = conveyorBelt
		log.Printf("[GameScene] Loaded conveyor belt animation (6 rows)")
	}

	// Note: Sun counter background is drawn procedurally for now
	// A dedicated image can be loaded here in the future if needed
	// Menu button resources are now loaded via ButtonFactory (ECS architecture)
}

// loadConveyorCardResources 加载传送带卡片渲染资源
// 必须在 ReanimSystem 初始化后调用（需要使用 RenderPlantIcon）
func (s *GameScene) loadConveyorCardResources() {
	// 加载卡片背景框
	cardBg, err := s.resourceManager.LoadImageByID(config.PlantCardBackgroundID)
	if err != nil {
		log.Printf("Warning: Failed to load conveyor card background: %v", err)
	} else {
		s.conveyorCardBackground = cardBg
		log.Printf("[GameScene] Loaded conveyor card background")
	}

	// 使用 RenderPlantIcon 渲染坚果图标
	wallnutIcon, err := entities.RenderPlantIcon(
		s.entityManager,
		s.resourceManager,
		s.reanimSystem,
		components.PlantWallnut,
	)
	if err != nil {
		log.Printf("Warning: Failed to render wallnut icon for conveyor: %v", err)
	} else {
		s.conveyorWallnutIcon = wallnutIcon
		log.Printf("[GameScene] Rendered wallnut icon for conveyor cards")
	}

	// 爆炸坚果使用相同的图标（后续可以添加红色染色）
	// 暂时复用普通坚果图标
	s.conveyorExplodeNutIcon = s.conveyorWallnutIcon
	log.Printf("[GameScene] Conveyor card resources loaded")
}

// loadSoddingResources loads sodding animation resources after level config is loaded.
// Story 8.2 QA改进：铺草皮动画资源加载
// Story 8.3: 添加奖励面板资源加载
//
// This method must be called AFTER the level configuration is loaded,
// because it depends on CurrentLevel.SodRowImage and CurrentLevel.ShowSoddingAnim.
func (s *GameScene) loadSoddingResources() {
	// Story 8.3: 加载 LoadingImages 资源组（包含按钮等 UI 资源）
	// 包含 IMAGE_SEEDCHOOSER_BUTTON 等资源
	if err := s.resourceManager.LoadResourceGroup("LoadingImages"); err != nil {
		log.Printf("Warning: Failed to load LoadingImages resources: %v", err)
	} else {
		log.Printf("[GameScene] 加载 UI 资源组成功 (LoadingImages)")
	}

	// Story 8.3: 加载奖励面板资源（延迟加载组）
	// 包含 AwardScreen_Back.jpg 等资源
	if err := s.resourceManager.LoadResourceGroup("DelayLoad_AwardScreen"); err != nil {
		log.Printf("Warning: Failed to load reward panel resources: %v", err)
	} else {
		log.Printf("[GameScene] 加载奖励面板资源组成功 (DelayLoad_AwardScreen)")
	}

	// 检查是否需要加载未铺草皮背景资源组
	if s.gameState.CurrentLevel != nil &&
		(s.gameState.CurrentLevel.BackgroundImage == "IMAGE_BACKGROUND1UNSODDED" ||
			s.gameState.CurrentLevel.SodRowImage != "") {
		if err := s.resourceManager.LoadResourceGroup("DelayLoad_BackgroundUnsodded"); err != nil {
			log.Printf("Warning: Failed to load BackgroundUnsodded resource group: %v", err)
		} else {
			log.Printf("[GameScene] 加载未铺草皮背景资源组成功")
		}
	}

	// 重新加载背景（如果需要切换到未铺草皮背景）
	// CRITICAL: 同样需要创建副本，避免修改缓存中的原始图片
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.BackgroundImage == "IMAGE_BACKGROUND1UNSODDED" {
		cachedBg, err := s.resourceManager.LoadImageByID("IMAGE_BACKGROUND1UNSODDED")
		if err != nil {
			log.Printf("Warning: Failed to load unsodded background: %v", err)
		} else {
			// 创建背景副本
			bounds := cachedBg.Bounds()
			bgCopy := ebiten.NewImage(bounds.Dx(), bounds.Dy())
			op := &ebiten.DrawImageOptions{}
			bgCopy.DrawImage(cachedBg, op)
			s.background = bgCopy

			// 重新计算摄像机边界
			bgWidth := bounds.Dx()
			s.maxCameraX = float64(bgWidth - WindowWidth)
			if s.maxCameraX < 0 {
				s.maxCameraX = 0
			}
			log.Printf("[GameScene] 切换到未铺草皮背景副本: (%dx%d)", bounds.Dx(), bounds.Dy())
		}
	}

	// Load sodding overlay images
	s.loadSoddingOverlayImages()

	// Pre-render sodded background
	s.preRenderSoddedBackground()
}

// loadSoddingOverlayImages loads sodding overlay images based on level configuration.
// Story 8.2 QA改进：加载草皮叠加图片（用于动画播放时的叠加渲染）
// 重构简化：所有叠加层X坐标从 0 开始，Y坐标对齐到草皮行位置
// - 启用行为连续3行（如 [2,3,4]）→ 使用 IMAGE_SOD3ROW（整体效果，无边缘）
// - 启用行为5行，动画行为 [1,5]（Level 1-4）→ 双背景叠加（IMAGE_BACKGROUND1）
// - 其他情况 → 使用 IMAGE_SOD1ROW（逐行渲染）
// - 两阶段渲染（Level 1-2）→ 初始使用 IMAGE_SOD1ROW，动画时使用 IMAGE_SOD3ROW
func (s *GameScene) loadSoddingOverlayImages() {
	if s.gameState.CurrentLevel == nil || !s.gameState.CurrentLevel.ShowSoddingAnim {
		return
	}

	enabledLanes := s.gameState.CurrentLevel.EnabledLanes
	animLanes := s.gameState.CurrentLevel.SoddingAnimLanes
	if len(animLanes) == 0 {
		animLanes = enabledLanes
	}

	// 检测启用行是否为连续3行
	isConsecutive3Rows := len(enabledLanes) == 3 &&
		enabledLanes[1] == enabledLanes[0]+1 &&
		enabledLanes[2] == enabledLanes[1]+1

	// 检测是否为 Level 1-4 场景（5行启用，动画行为 [1,5]）
	isLevel14Pattern := len(enabledLanes) == 5 &&
		len(animLanes) == 2 &&
		animLanes[0] == 1 && animLanes[1] == 5

	// 检测是否需要两阶段渲染（配置了 sodRowImageAnim）
	hasTwoStageRendering := s.gameState.CurrentLevel.SodRowImageAnim != ""

	if isLevel14Pattern && hasTwoStageRendering && s.gameState.CurrentLevel.SodRowImageAnim == "IMAGE_BACKGROUND1" {
		// Level 1-4: 双背景叠加模式，使用 sodRowImageAnim="IMAGE_BACKGROUND1" 作为叠加层
		s.loadLevel14Background()
	} else if isLevel14Pattern {
		// Level 1-4（旧版兼容）: 双背景叠加模式，加载 IMAGE_BACKGROUND1 作为叠加层
		s.loadLevel14BackgroundLegacy()
	} else if hasTwoStageRendering && isConsecutive3Rows {
		// 两阶段渲染模式（Level 1-2）
		s.loadTwoStageSoddingImages(enabledLanes)
	} else if isConsecutive3Rows {
		// 连续3行：使用 IMAGE_SOD3ROW（整体草皮，无边缘分界线）
		s.loadConsecutive3RowsSodding(enabledLanes)
	} else {
		// 其他情况：使用 IMAGE_SOD1ROW（单行草皮）
		s.loadSingleRowSodding(animLanes)
	}

	// 启动铺草皮动画
	s.soddingAnimDelay = s.gameState.CurrentLevel.SoddingAnimDelay
	s.soddingAnimStarted = false
	s.soddingAnimTimer = 0
	log.Printf("[GameScene] 设置铺草皮动画延迟: %.1f 秒", s.soddingAnimDelay)
}

// loadLevel14Background loads Level 1-4 background (sodded background overlay).
func (s *GameScene) loadLevel14Background() {
	log.Printf("[GameScene] 检测到 Level 1-4 双背景叠加模式：sodRowImageAnim=%s", s.gameState.CurrentLevel.SodRowImageAnim)
	log.Printf("[GameScene] 底层=未铺草皮+预渲染(IMAGE_SOD3ROW), 叠加层=IMAGE_BACKGROUND1")

	// 加载已铺草皮完整背景（IMAGE_BACKGROUND1）作为叠加层
	soddedBg, err := s.resourceManager.LoadImageByID("IMAGE_BACKGROUND1")
	if err != nil {
		log.Printf("Warning: Failed to load IMAGE_BACKGROUND1: %v", err)
	} else {
		s.soddedBackground = soddedBg
		log.Printf("[GameScene] ✅ 加载已铺草皮背景作为叠加层: IMAGE_BACKGROUND1")
	}

	// 重构简化：叠加背景从 (0,0) 开始（与底层背景完全对齐）
	s.sodOverlayX = 0
	s.sodOverlayY = 0
	log.Printf("[GameScene] 双背景叠加模式：叠加层从 (0,0) 开始")
}

// loadLevel14BackgroundLegacy loads Level 1-4 background (legacy version).
func (s *GameScene) loadLevel14BackgroundLegacy() {
	log.Printf("[GameScene] 检测到 Level 1-4 模式：双背景叠加（底层=未铺草皮+预渲染，叠加层=IMAGE_BACKGROUND1）")

	// 加载已铺草皮完整背景（IMAGE_BACKGROUND1）
	soddedBg, err := s.resourceManager.LoadImageByID("IMAGE_BACKGROUND1")
	if err != nil {
		log.Printf("Warning: Failed to load IMAGE_BACKGROUND1: %v", err)
	} else {
		s.soddedBackground = soddedBg
		log.Printf("[GameScene] ✅ 加载已铺草皮背景: IMAGE_BACKGROUND1")
	}

	// 重构简化：叠加背景从 (0,0) 开始（与底层背景完全对齐）
	s.sodOverlayX = 0
	s.sodOverlayY = 0
	log.Printf("[GameScene] 双背景叠加模式：叠加层从 (0,0) 开始")
}

// loadTwoStageSoddingImages loads two-stage sodding images (Level 1-2).
// 阶段1（初始化）：使用 sodRowImage（IMAGE_SOD1ROW）预渲染指定行
// 阶段2（动画播放）：使用 sodRowImageAnim（IMAGE_SOD3ROW）叠加渲染
func (s *GameScene) loadTwoStageSoddingImages(enabledLanes []int) {
	log.Printf("[GameScene] 检测到两阶段渲染模式：初始=%s, 动画=%s",
		s.gameState.CurrentLevel.SodRowImage, s.gameState.CurrentLevel.SodRowImageAnim)

	// 加载动画阶段使用的草皮图片（IMAGE_SOD3ROW）
	sod3RowImage, err := s.resourceManager.LoadImageWithAlphaMask(
		"assets/images/sod3row.jpg",
		"assets/images/sod3row_.png",
	)
	if err != nil {
		log.Printf("Warning: Failed to composite sod3row image: %v", err)
		return
	}

	s.sodRowImage = sod3RowImage
	log.Printf("[GameScene] ✅ 合成草皮叠加图片 (RGB + Alpha): IMAGE_SOD3ROW (动画阶段)")

	// 性能优化：缓存草皮图片尺寸
	sodBounds := sod3RowImage.Bounds()
	s.sodWidth = sodBounds.Dx()
	s.sodHeight = sodBounds.Dy()

	// 计算草皮叠加层Y坐标（需要与预渲染时的位置一致）
	// 两阶段渲染：使用中间行的中心位置
	// Story 8.2.1 修复：sodOverlayY应该使用实际的草皮位置（居中+偏移）
	middleLane := enabledLanes[len(enabledLanes)/2]
	rowCenterY := config.GridWorldStartY + float64(middleLane-1)*config.CellHeight + config.CellHeight/2.0
	sodHeight := float64(s.sodHeight)
	sodOverlayY := rowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY

	// Story 8.2.1 修复：sodOverlayX应该使用实际的草皮位置（GridWorldStartX + SodOverlayOffsetX）
	// 因为草皮在预渲染时被放置在这个位置，闪烁效果需要对齐
	s.sodOverlayX = config.GridWorldStartX + config.SodOverlayOffsetX
	s.sodOverlayY = sodOverlayY
	log.Printf("[GameScene] 草皮叠加层: 位置(%.1f, %.1f) 尺寸(%dx%d)", s.sodOverlayX, sodOverlayY, s.sodWidth, s.sodHeight)
}

// loadConsecutive3RowsSodding loads consecutive 3 rows sodding image (IMAGE_SOD3ROW).
func (s *GameScene) loadConsecutive3RowsSodding(enabledLanes []int) {
	sod3RowImage, err := s.resourceManager.LoadImageWithAlphaMask(
		"assets/images/sod3row.jpg",
		"assets/images/sod3row_.png",
	)
	if err != nil {
		log.Printf("Warning: Failed to composite sod row image: %v", err)
		return
	}

	s.sodRowImage = sod3RowImage
	log.Printf("[GameScene] ✅ 合成草皮叠加图片 (RGB + Alpha): IMAGE_SOD3ROW (启用行: %v)", enabledLanes)

	// 性能优化：缓存草皮图片尺寸
	sodBounds := sod3RowImage.Bounds()
	s.sodWidth = sodBounds.Dx()
	s.sodHeight = sodBounds.Dy()

	// 计算草皮叠加层Y坐标（需要与预渲染时的位置一致）
	// 连续3行：使用中间行的中心位置
	// Story 8.2.1 修复：sodOverlayY应该使用实际的草皮位置（居中+偏移）
	middleLane := enabledLanes[len(enabledLanes)/2]
	rowCenterY := config.GridWorldStartY + float64(middleLane-1)*config.CellHeight + config.CellHeight/2.0
	sodHeight := float64(s.sodHeight)
	sodOverlayY := rowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY

	// Story 8.2.1 修复：sodOverlayX应该使用实际的草皮位置（GridWorldStartX + SodOverlayOffsetX）
	// 因为草皮在预渲染时被放置在这个位置，闪烁效果需要对齐
	s.sodOverlayX = config.GridWorldStartX + config.SodOverlayOffsetX
	s.sodOverlayY = sodOverlayY
	log.Printf("[GameScene] 草皮叠加层: 位置(%.1f, %.1f) 尺寸(%dx%d)", s.sodOverlayX, sodOverlayY, s.sodWidth, s.sodHeight)
}

// loadSingleRowSodding loads single row sodding image (IMAGE_SOD1ROW).
func (s *GameScene) loadSingleRowSodding(animLanes []int) {
	sod1RowImage, err := s.resourceManager.LoadImageWithAlphaMask(
		"assets/images/sod1row.jpg",
		"assets/images/sod1row_.png",
	)
	if err != nil {
		log.Printf("Warning: Failed to composite sod row image: %v", err)
		return
	}

	s.sodRowImage = sod1RowImage
	log.Printf("[GameScene] ✅ 合成草皮叠加图片 (RGB + Alpha): IMAGE_SOD1ROW (启用行: %v)", animLanes)

	// 性能优化：缓存草皮图片尺寸
	sodBounds := sod1RowImage.Bounds()
	s.sodWidth = sodBounds.Dx()
	s.sodHeight = sodBounds.Dy()

	// 计算草皮叠加层Y坐标（需要与预渲染时的位置一致）
	// 单行模式下，使用第一个动画行的位置
	// Story 8.2.1 修复：sodOverlayY应该使用实际的草皮位置（居中+偏移）
	firstAnimLane := animLanes[0]
	rowCenterY := config.GridWorldStartY + float64(firstAnimLane-1)*config.CellHeight + config.CellHeight/2.0
	sodHeight := float64(s.sodHeight)
	sodOverlayY := rowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY

	// Story 8.2.1 修复：sodOverlayX应该使用实际的草皮位置（GridWorldStartX + SodOverlayOffsetX）
	// 因为草皮在预渲染时被放置在这个位置，闪烁效果需要对齐
	s.sodOverlayX = config.GridWorldStartX + config.SodOverlayOffsetX
	s.sodOverlayY = sodOverlayY
	log.Printf("[GameScene] 草皮叠加层: 位置(%.1f, %.1f) 尺寸(%dx%d)", s.sodOverlayX, sodOverlayY, s.sodWidth, s.sodHeight)
}

// preRenderSoddedBackground pre-renders sodded background for overlay rendering.
// Story 8.6 QA修正 + 统一草皮渲染重构：
// 为所有启用的行预渲染草皮到背景副本，用于双背景叠加渲染
//
// 设计思路（两阶段渲染模式 - Level 1-2）：
// - 底层背景：未铺草皮背景 + preSoddedLanes 草皮（IMAGE_SOD1ROW）
// - 叠加层：未铺草皮背景 + 所有启用行草皮（sodRowImageAnim 如 IMAGE_SOD3ROW）
// - 动画播放时：叠加层渐进显示，覆盖底层的 IMAGE_SOD1ROW，展现完整的 IMAGE_SOD3ROW
//
// Level 1-3: preSoddedLanes=[2,3,4], ShowSoddingAnim=false
// - 直接渲染3行草皮到背景，无需动画
func (s *GameScene) preRenderSoddedBackground() {
	if s.gameState.CurrentLevel == nil || s.background == nil {
		return
	}

	if !s.gameState.CurrentLevel.ShowSoddingAnim && len(s.gameState.CurrentLevel.PreSoddedLanes) == 0 {
		return
	}

	enabledLanes := s.gameState.CurrentLevel.EnabledLanes
	preSoddedLanes := s.gameState.CurrentLevel.PreSoddedLanes
	hasTwoStageRendering := s.gameState.CurrentLevel.SodRowImageAnim != ""

	// 步骤1：预渲染底层背景的草皮（preSoddedLanes）
	s.preRenderBaseLayerSodding(preSoddedLanes)

	// 步骤2：预渲染叠加层背景（用于动画时渐进显示）
	s.preRenderOverlayLayerSodding(enabledLanes, preSoddedLanes, hasTwoStageRendering)
}

// preRenderBaseLayerSodding pre-renders base layer sodding (preSoddedLanes).
func (s *GameScene) preRenderBaseLayerSodding(preSoddedLanes []int) {
	if len(preSoddedLanes) == 0 {
		return
	}

	// 检查预铺行是否是连续的3行（第2,3,4行）
	isConsecutive3Rows := len(preSoddedLanes) == 3 &&
		preSoddedLanes[0] == 2 && preSoddedLanes[1] == 3 && preSoddedLanes[2] == 4

	// 根据配置的 sodRowImage 选择使用的草皮图片
	var sodRowRGB *ebiten.Image
	var sodImageID string
	var err error

	if s.gameState.CurrentLevel.SodRowImage == "IMAGE_SOD3ROW" && isConsecutive3Rows {
		// 使用 IMAGE_SOD3ROW（3行整体草皮）
		sodRowRGB, err = s.resourceManager.LoadImageWithAlphaMask(
			"assets/images/sod3row.jpg",
			"assets/images/sod3row_.png",
		)
		sodImageID = "IMAGE_SOD3ROW"
	} else {
		// 默认使用 IMAGE_SOD1ROW（单行草皮）
		sodRowRGB, err = s.resourceManager.LoadImageWithAlphaMask(
			"assets/images/sod1row.jpg",
			"assets/images/sod1row_.png",
		)
		sodImageID = "IMAGE_SOD1ROW"
	}

	if err != nil {
		log.Printf("[GameScene] Error: 无法加载草皮图片 %s: %v", sodImageID, err)
		return
	}

	log.Printf("[GameScene] 预渲染底层背景草皮: 预铺行=%v (使用 %s)", preSoddedLanes, sodImageID)

	// 根据草皮图片类型选择渲染方式
	if sodImageID == "IMAGE_SOD3ROW" && isConsecutive3Rows {
		s.renderConsecutive3RowsToBackground(sodRowRGB, preSoddedLanes)
	} else {
		s.renderSingleRowsToBackground(sodRowRGB, preSoddedLanes)
	}
}

// renderConsecutive3RowsToBackground renders consecutive 3 rows sodding to background.
func (s *GameScene) renderConsecutive3RowsToBackground(sodRowRGB *ebiten.Image, lanes []int) {
	// IMAGE_SOD3ROW：整体渲染3行草皮（无需循环）
	// 图片覆盖第2,3,4行，图片中心应该对齐到第3行（中间行）的中心
	sodBounds := sodRowRGB.Bounds()
	sodHeight := float64(sodBounds.Dy())

	// 计算中间行（第3行）的中心Y坐标
	middleLane := lanes[1] // 第3行（索引1）
	middleRowCenterY := config.GridWorldStartY + float64(middleLane-1)*config.CellHeight + config.CellHeight/2.0

	// 草皮Y坐标 = 中间行中心 - 草皮高度的一半 + 偏移
	// 这样图片的中心对齐到第3行的中心，覆盖第2,3,4行
	dstY := middleRowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY

	// 草皮X坐标
	dstX := config.GridWorldStartX + config.SodOverlayOffsetX

	// 整体绘制到底层背景
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(dstX, dstY)
	s.background.DrawImage(sodRowRGB, op)

	log.Printf("[GameScene] ✅ 底层背景预渲染第 %v 行草皮 (IMAGE_SOD3ROW 整体): 中心对齐第%d行, 位置(%.1f,%.1f)",
		lanes, middleLane, dstX, dstY)
}

// renderSingleRowsToBackground renders single rows sodding to background.
func (s *GameScene) renderSingleRowsToBackground(sodRowRGB *ebiten.Image, lanes []int) {
	// IMAGE_SOD1ROW：逐行渲染单行草皮
	for _, lane := range lanes {
		sodBounds := sodRowRGB.Bounds()
		sodHeight := float64(sodBounds.Dy())

		// 计算目标行的中心Y坐标
		rowCenterY := config.GridWorldStartY + float64(lane-1)*config.CellHeight + config.CellHeight/2.0

		// 草皮Y坐标 = 行中心 - 草皮高度的一半 + 偏移
		dstY := rowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY

		// 草皮X坐标
		dstX := config.GridWorldStartX + config.SodOverlayOffsetX

		// 绘制到底层背景
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(dstX, dstY)
		s.background.DrawImage(sodRowRGB, op)

		log.Printf("[GameScene] ✅ 底层背景预渲染第 %d 行草皮 (IMAGE_SOD1ROW): 位置(%.1f,%.1f)", lane, dstX, dstY)
	}
}

// preRenderOverlayLayerSodding pre-renders overlay layer sodding.
func (s *GameScene) preRenderOverlayLayerSodding(enabledLanes, preSoddedLanes []int, hasTwoStageRendering bool) {
	// 创建新的背景图片副本（叠加层）
	bgBounds := s.background.Bounds()
	newBackground := ebiten.NewImage(bgBounds.Dx(), bgBounds.Dy())

	// 1. 绘制原始背景（现在已包含 preSoddedLanes 草皮）
	op := &ebiten.DrawImageOptions{}
	newBackground.DrawImage(s.background, op)

	// 2. 预渲染叠加层草皮
	// 设计原理：
	// - preSoddedLanes：控制底层背景预渲染哪些行（动画开始前就可见）
	// - 叠加层：始终包含所有启用行的草皮，用于动画时逐渐显示
	//
	// 示例：
	// Level 1-1: preSoddedLanes=[], enabledLanes=[3]
	//   → 底层无草皮，叠加层有第3行草皮，通过裁剪逐渐显示
	// Level 1-2: preSoddedLanes=[2,4], enabledLanes=[2,3,4]
	//   → 底层有2/4行草皮，叠加层有2/3/4行草皮，动画显示第3行
	lanesToPreRender := enabledLanes

	if len(lanesToPreRender) == 0 {
		return
	}

	// 检查是否是连续的3行
	isConsecutive3Rows := len(lanesToPreRender) == 3 &&
		lanesToPreRender[1] == lanesToPreRender[0]+1 &&
		lanesToPreRender[2] == lanesToPreRender[1]+1

	log.Printf("[GameScene] 预渲染叠加层草皮: 启用行=%v, 预铺行=%v, 实际预渲染=%v, 两阶段模式=%v",
		enabledLanes, preSoddedLanes, lanesToPreRender, hasTwoStageRendering)

	// 根据行数和两阶段模式选择渲染方式
	if hasTwoStageRendering && isConsecutive3Rows {
		s.renderTwoStageOverlay(newBackground, lanesToPreRender)
	} else if isConsecutive3Rows {
		s.renderConsecutive3RowsOverlay(newBackground, lanesToPreRender)
	} else {
		s.renderSingleRowsOverlay(newBackground, lanesToPreRender)
	}

	// 3. 保存预渲染背景副本（用于草皮叠加渲染）
	s.preSoddedImage = newBackground

	log.Printf("[GameScene] ✅ 创建预渲染背景副本用于草皮叠加 (preSoddedLanes: %v, 两阶段模式: %v)", preSoddedLanes, hasTwoStageRendering)
}

// renderTwoStageOverlay renders two-stage overlay (IMAGE_SOD3ROW).
func (s *GameScene) renderTwoStageOverlay(newBackground *ebiten.Image, lanesToPreRender []int) {
	// 两阶段模式：叠加层使用 IMAGE_SOD3ROW
	sod3RowRGB, err := s.resourceManager.LoadImageWithAlphaMask(
		"assets/images/sod3row.jpg",
		"assets/images/sod3row_.png",
	)
	if err != nil {
		log.Printf("[GameScene] Error: 无法加载3行草皮图片: %v", err)
		return
	}

	// 计算中间行的中心Y坐标
	middleLane := lanesToPreRender[1] // 中间行
	rowCenterY := config.GridWorldStartY + float64(middleLane-1)*config.CellHeight + config.CellHeight/2.0

	// 3行草皮图片的高度
	sodBounds := sod3RowRGB.Bounds()
	sodHeight := float64(sodBounds.Dy())

	// 草皮Y坐标 = 中间行中心 - 草皮高度的一半 + 偏移
	dstY := rowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY

	// 草皮X坐标
	dstX := config.GridWorldStartX + config.SodOverlayOffsetX

	// 一次性绘制3行草皮到叠加层
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(dstX, dstY)
	newBackground.DrawImage(sod3RowRGB, op)

	log.Printf("[GameScene] ✅ 叠加层预渲染第 %v 行草皮 (IMAGE_SOD3ROW): 位置(%.1f,%.1f)", lanesToPreRender, dstX, dstY)
}

// renderConsecutive3RowsOverlay renders consecutive 3 rows overlay (IMAGE_SOD3ROW).
func (s *GameScene) renderConsecutive3RowsOverlay(newBackground *ebiten.Image, lanesToPreRender []int) {
	// 连续3行但非两阶段模式：使用 IMAGE_SOD3ROW
	sod3RowRGB, err := s.resourceManager.LoadImageWithAlphaMask(
		"assets/images/sod3row.jpg",
		"assets/images/sod3row_.png",
	)
	if err != nil {
		log.Printf("[GameScene] Error: 无法加载3行草皮图片: %v", err)
		return
	}

	middleLane := lanesToPreRender[1]
	rowCenterY := config.GridWorldStartY + float64(middleLane-1)*config.CellHeight + config.CellHeight/2.0
	sodBounds := sod3RowRGB.Bounds()
	sodHeight := float64(sodBounds.Dy())
	dstY := rowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY
	dstX := config.GridWorldStartX + config.SodOverlayOffsetX

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(dstX, dstY)
	newBackground.DrawImage(sod3RowRGB, op)

	log.Printf("[GameScene] ✅ 使用 IMAGE_SOD3ROW 一次性预渲染第 %v 行草皮: 背景位置(%.1f,%.1f)", lanesToPreRender, dstX, dstY)
}

// renderSingleRowsOverlay renders single rows overlay (IMAGE_SOD1ROW).
func (s *GameScene) renderSingleRowsOverlay(newBackground *ebiten.Image, lanesToPreRender []int) {
	// 使用 IMAGE_SOD1ROW 循环渲染每行
	sod1RowRGB, err := s.resourceManager.LoadImageWithAlphaMask(
		"assets/images/sod1row.jpg",
		"assets/images/sod1row_.png",
	)
	if err != nil {
		log.Printf("[GameScene] Error: 无法加载单行草皮图片: %v", err)
		return
	}

	// 为每个需要预渲染的行绘制单行草皮
	for _, lane := range lanesToPreRender {
		sodBounds := sod1RowRGB.Bounds()
		sodHeight := float64(sodBounds.Dy())
		rowCenterY := config.GridWorldStartY + float64(lane-1)*config.CellHeight + config.CellHeight/2.0
		dstY := rowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY
		dstX := config.GridWorldStartX + config.SodOverlayOffsetX

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(dstX, dstY)
		newBackground.DrawImage(sod1RowRGB, op)

		log.Printf("[GameScene] ✅ 使用 IMAGE_SOD1ROW 预渲染第 %d 行草皮: 背景位置(%.1f,%.1f)", lane, dstX, dstY)
	}
}
