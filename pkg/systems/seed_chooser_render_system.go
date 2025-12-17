package systems

import (
	"image/color"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// SeedChooserRenderSystem 选卡界面渲染系统
// Story 8.12: 负责渲染关卡 1-8 及以后的选卡界面
// 重构：复用 entities.RenderPlantCard() 统一渲染逻辑
type SeedChooserRenderSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager
	reanimSystem    *ReanimSystem
	levelConfig     *config.LevelConfig
	inputSystem     *SeedChooserInputSystem // 输入系统引用（依赖注入）

	// 图片资源
	panelBackground *ebiten.Image // 选卡面板背景
	silhouetteImage *ebiten.Image // 空卡槽剪影
	buttonNormal    *ebiten.Image // 按钮正常状态
	buttonDisabled  *ebiten.Image // 按钮禁用状态
	buttonGlow      *ebiten.Image // 按钮发光效果

	// 字体
	titleFont     *text.GoTextFace       // 标题字体
	sunFont       *text.GoTextFace       // 阳光数字字体
	sunFontSource *text.GoTextFaceSource // 字体源（用于 RenderPlantCard）
	sunFontSize   float64                // 字体大小

	// 植物卡片组件（复用 PlantCardComponent 进行渲染）
	plantCards      map[string]*components.PlantCardComponent // 植物ID -> 卡片组件
	availablePlants []string                                  // 可用植物列表（按顺序）

	// Tooltip 数据（植物名称和描述）
	plantNames    map[string]string // 植物ID -> 植物名称
	plantTooltips map[string]string // 植物ID -> 植物描述
	zombieNames   map[string]string // 僵尸UnitID -> 僵尸名称
}

// NewSeedChooserRenderSystem 创建选卡界面渲染系统
func NewSeedChooserRenderSystem(
	em *ecs.EntityManager,
	gs *game.GameState,
	rm *game.ResourceManager,
	rs *ReanimSystem,
	levelConfig *config.LevelConfig,
	titleFont, sunFont *text.GoTextFace,
) *SeedChooserRenderSystem {
	var fontSource *text.GoTextFaceSource
	var fontSize float64 = 20.0
	if sunFont != nil {
		fontSource = sunFont.Source
		fontSize = sunFont.Size
	}

	system := &SeedChooserRenderSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		reanimSystem:    rs,
		levelConfig:     levelConfig,
		titleFont:       titleFont,
		sunFont:         sunFont,
		sunFontSource:   fontSource,
		sunFontSize:     fontSize,
		plantCards:      make(map[string]*components.PlantCardComponent),
		plantNames:      make(map[string]string),
		plantTooltips:   make(map[string]string),
		zombieNames:     make(map[string]string),
	}

	// 加载图片资源
	system.loadResources()

	// 预创建植物卡片组件（复用 PlantCardComponent）
	system.preCreatePlantCards()

	// 加载植物 Tooltip 数据
	system.loadPlantTooltips()

	// 加载僵尸名称数据
	system.loadZombieNames()

	log.Printf("[SeedChooserRenderSystem] 初始化完成，可用植物: %v", system.availablePlants)

	return system
}

// loadResources 加载所有图片资源
func (s *SeedChooserRenderSystem) loadResources() {
	var err error

	// 加载面板背景
	s.panelBackground, err = s.resourceManager.LoadImage("assets/images/SeedChooser_Background.png")
	if err != nil {
		log.Printf("[SeedChooserRenderSystem] 警告: 加载面板背景失败: %v", err)
	}

	// 加载空卡槽剪影
	s.silhouetteImage, err = s.resourceManager.LoadImage("assets/images/SeedPacketSilhouette.png")
	if err != nil {
		log.Printf("[SeedChooserRenderSystem] 警告: 加载卡槽剪影失败: %v", err)
	}

	// 加载按钮图片
	s.buttonNormal, err = s.resourceManager.LoadImage("assets/images/SeedChooser_Button.png")
	if err != nil {
		log.Printf("[SeedChooserRenderSystem] 警告: 加载按钮图片失败: %v", err)
	}

	s.buttonDisabled, err = s.resourceManager.LoadImage("assets/images/SeedChooser_Button_Disabled.png")
	if err != nil {
		log.Printf("[SeedChooserRenderSystem] 警告: 加载禁用按钮图片失败: %v", err)
	}

	s.buttonGlow, err = s.resourceManager.LoadImage("assets/images/SeedChooser_Button_Glow.png")
	if err != nil {
		log.Printf("[SeedChooserRenderSystem] 警告: 加载按钮发光图片失败: %v", err)
	}
}

// preCreatePlantCards 预创建所有可用植物的卡片组件
// 复用 PlantCardComponent 结构，确保与游戏中卡片渲染完全一致
func (s *SeedChooserRenderSystem) preCreatePlantCards() {
	// 获取可用植物列表
	// 优先使用关卡配置的植物列表（当前关卡解锁的植物）
	if s.levelConfig != nil && len(s.levelConfig.AvailablePlants) > 0 {
		s.availablePlants = s.levelConfig.AvailablePlants
	} else {
		// 无关卡配置时，从植物注册表获取所有植物（用于验证程序等场景）
		registeredPlants := config.GetAllPlants()
		s.availablePlants = make([]string, len(registeredPlants))
		for i, plant := range registeredPlants {
			s.availablePlants[i] = plant.ID
		}
	}

	// 加载卡片背景图（所有卡片共用）
	backgroundImg, err := s.resourceManager.LoadImageByID(config.PlantCardBackgroundID)
	if err != nil {
		log.Printf("[SeedChooserRenderSystem] 警告: 加载卡片背景失败: %v", err)
	}

	// 为每个植物创建 PlantCardComponent（使用植物注册表查询）
	for _, plantID := range s.availablePlants {
		// 从植物注册表获取植物类型
		plantType := config.PlantIDToType(plantID)
		if plantType == components.PlantUnknown {
			log.Printf("[SeedChooserRenderSystem] 未知植物类型: %s", plantID)
			continue
		}

		// 渲染植物图标（复用 entities.RenderPlantIcon）
		icon, err := entities.RenderPlantIcon(s.entityManager, s.resourceManager, s.reanimSystem, plantType)
		if err != nil {
			log.Printf("[SeedChooserRenderSystem] 渲染植物图标失败 %s: %v", plantID, err)
			continue
		}

		// 创建 PlantCardComponent（用于复用 RenderPlantCard）
		// 从植物注册表获取阳光消耗
		s.plantCards[plantID] = &components.PlantCardComponent{
			PlantType:        plantType,
			SunCost:          config.GetPlantSunCost(plantID),
			BackgroundImage:  backgroundImg,
			PlantIconTexture: icon,
			CardScale:        config.PlantCardScale,
			Alpha:            1.0,
			IsAvailable:      true, // 选卡界面默认可用
		}
	}

	log.Printf("[SeedChooserRenderSystem] 预创建了 %d 个植物卡片组件", len(s.plantCards))
}

// Draw 渲染选卡界面
// 只在选卡阶段激活时渲染
func (s *SeedChooserRenderSystem) Draw(screen *ebiten.Image) {
	// 检查是否处于选卡阶段
	if !s.gameState.IsSeedSelectionActive() {
		return
	}

	// 1. 渲染面板背景
	s.drawPanelBackground(screen)

	// 2. 渲染标题
	s.drawTitle(screen)

	// 3. 渲染植物仓库（可选植物列表）
	s.drawInventory(screen)

	// 4. 渲染顶部卡槽
	s.drawSlots(screen)

	// 5. 渲染"一起摇滚吧!"按钮
	s.drawButton(screen)

	// 6. 渲染飞行中的卡片
	s.DrawFlyingCards(screen)

	// 7. 渲染悬停提示框（在最上层）
	s.DrawTooltip(screen)
}

// drawPanelBackground 渲染选卡面板背景
func (s *SeedChooserRenderSystem) drawPanelBackground(screen *ebiten.Image) {
	if s.panelBackground == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(config.SeedChooserPanelX, config.SeedChooserPanelY)
	screen.DrawImage(s.panelBackground, op)
}

// drawTitle 渲染标题文字
func (s *SeedChooserRenderSystem) drawTitle(screen *ebiten.Image) {
	if s.titleFont == nil {
		return
	}

	titleText := config.SeedChooserTitle

	// 计算文本宽度用于居中
	textWidth := text.Advance(titleText, s.titleFont)

	// 标题位置（居中）
	x := config.SeedChooserTitleX - textWidth/2
	y := float64(config.SeedChooserTitleY)

	// 绘制阴影
	shadowOp := &text.DrawOptions{}
	shadowOp.GeoM.Translate(x+2, y+2)
	shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 180})
	text.Draw(screen, titleText, s.titleFont, shadowOp)

	// 绘制文字（金黄色）
	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(x, y)
	textOp.ColorScale.ScaleWithColor(color.RGBA{255, 220, 100, 255})
	text.Draw(screen, titleText, s.titleFont, textOp)
}

// SeedInventoryRowCount 仓库行数
const SeedInventoryRowCount = 5

// drawInventory 渲染植物仓库（可选植物列表）
// Story 8.12: 先渲染5行8列空卡轮廓剪影作为背景，然后渲染植物卡片
func (s *SeedChooserRenderSystem) drawInventory(screen *ebiten.Image) {
	selectedPlants := s.gameState.GetSeedChooserPlants()
	isFull := s.gameState.IsSeedChooserFull()

	// 第一遍：渲染5行8列的空卡轮廓剪影作为背景（40个位置）
	for row := 0; row < SeedInventoryRowCount; row++ {
		for col := 0; col < config.SeedInventoryCardsPerRow; col++ {
			x := float64(config.SeedInventoryStartX) + float64(col)*(config.SeedInventoryCardWidth+config.SeedInventoryCardSpacingX)
			y := float64(config.SeedInventoryStartY) + float64(row)*(config.SeedInventoryCardHeight+config.SeedInventoryCardSpacingY)

			// 渲染空卡轮廓剪影背景
			s.drawInventorySilhouette(screen, x, y)
		}
	}

	// 第二遍：渲染植物卡片（在剪影之上）
	for i, plantID := range s.availablePlants {
		// 计算卡片位置
		row := i / config.SeedInventoryCardsPerRow
		col := i % config.SeedInventoryCardsPerRow
		x := float64(config.SeedInventoryStartX) + float64(col)*(config.SeedInventoryCardWidth+config.SeedInventoryCardSpacingX)
		y := float64(config.SeedInventoryStartY) + float64(row)*(config.SeedInventoryCardHeight+config.SeedInventoryCardSpacingY)

		// 检查是否已被选中（包括在 GameState 中和在待处理列表中）
		isSelected := s.isPlantSelected(plantID, selectedPlants) || s.isPlantPending(plantID)

		// 渲染卡片
		s.drawPlantCard(screen, plantID, x, y, isSelected, isFull)
	}
}

// isPlantPending 检查植物是否在待处理列表中（正在飞向卡槽）
func (s *SeedChooserRenderSystem) isPlantPending(plantID string) bool {
	if s.inputSystem == nil {
		return false
	}
	return s.inputSystem.IsPlantSelectedOrPending(plantID) && !s.gameState.IsSeedChooserPlantSelected(plantID)
}

// drawInventorySilhouette 渲染仓库位置的空卡轮廓剪影
func (s *SeedChooserRenderSystem) drawInventorySilhouette(screen *ebiten.Image, x, y float64) {
	if s.silhouetteImage == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	// 调整剪影大小以适应仓库卡片尺寸
	silhouetteBounds := s.silhouetteImage.Bounds()
	scaleX := config.SeedInventoryCardWidth / float64(silhouetteBounds.Dx())
	scaleY := config.SeedInventoryCardHeight / float64(silhouetteBounds.Dy())
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(x, y)
	screen.DrawImage(s.silhouetteImage, op)
}

// drawPlantCard 渲染单张植物卡片
// Story 8.12 重构: 完全复用 entities.RenderPlantCard() 统一渲染逻辑
func (s *SeedChooserRenderSystem) drawPlantCard(screen *ebiten.Image, plantID string, x, y float64, isSelected, isFull bool) {
	// 获取植物卡片组件
	card, ok := s.plantCards[plantID]
	if !ok {
		return
	}

	// 渲染卡片
	entities.RenderPlantCard(screen, card, x, y, s.sunFontSource, s.sunFontSize)

	// 已选中的卡片需要绘制禁用遮罩（使用与游戏中相同的渲染方式）
	if isSelected {
		s.drawDisabledOverlay(screen, x, y, config.SeedInventoryCardWidth, config.SeedInventoryCardHeight)
	}
}

// drawDisabledOverlay 绘制禁用遮罩（与游戏中卡片禁用状态一致）
func (s *SeedChooserRenderSystem) drawDisabledOverlay(screen *ebiten.Image, x, y, width, height float64) {
	intWidth := int(width)
	intHeight := int(height)
	if intWidth <= 0 || intHeight <= 0 {
		return
	}

	// 创建浅灰色遮罩，透明度120（与 plant_card_factory.go 中的 renderEffectMask 一致）
	mask := ebiten.NewImage(intWidth, intHeight)
	mask.Fill(color.RGBA{0, 0, 0, 120})

	maskOp := &ebiten.DrawImageOptions{}
	maskOp.GeoM.Translate(x, y)
	screen.DrawImage(mask, maskOp)
}

// drawSlots 渲染顶部卡槽
func (s *SeedChooserRenderSystem) drawSlots(screen *ebiten.Image) {
	selectedPlants := s.gameState.GetSeedChooserPlants()

	for i := 0; i < config.SeedSlotCount; i++ {
		x := float64(config.SeedSlotStartX) + float64(i)*(config.SeedSlotWidth+config.SeedSlotSpacing)
		y := float64(config.SeedSlotStartY)

		if i < len(selectedPlants) {
			// 已选中的卡片
			plantID := selectedPlants[i]
			s.drawSlotCard(screen, plantID, x, y)
		} else {
			// 空卡槽剪影
			s.drawSlotSilhouette(screen, x, y)
		}
	}
}

// drawSlotCard 渲染卡槽中的植物卡片
// Story 8.12 重构: 完全复用 entities.RenderPlantCard() 统一渲染逻辑
func (s *SeedChooserRenderSystem) drawSlotCard(screen *ebiten.Image, plantID string, x, y float64) {
	// 获取植物卡片组件
	card, ok := s.plantCards[plantID]
	if !ok {
		return
	}

	// 卡槽中的卡片正常渲染
	entities.RenderPlantCard(screen, card, x, y, s.sunFontSource, s.sunFontSize)
}

// drawSlotSilhouette 渲染空卡槽剪影
func (s *SeedChooserRenderSystem) drawSlotSilhouette(screen *ebiten.Image, x, y float64) {
	if s.silhouetteImage == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	// 调整剪影大小以适应卡槽
	silhouetteBounds := s.silhouetteImage.Bounds()
	scaleX := config.SeedSlotWidth / float64(silhouetteBounds.Dx())
	scaleY := config.SeedSlotHeight / float64(silhouetteBounds.Dy())
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(x, y)
	screen.DrawImage(s.silhouetteImage, op)
}

// drawButton 渲染"一起摇滚吧!"按钮
func (s *SeedChooserRenderSystem) drawButton(screen *ebiten.Image) {
	isFull := s.gameState.IsSeedChooserFull()

	// 选择按钮图片
	var buttonImg *ebiten.Image
	if isFull {
		buttonImg = s.buttonNormal
	} else {
		buttonImg = s.buttonDisabled
	}

	if buttonImg == nil {
		return
	}

	// 计算按钮位置（居中）
	buttonBounds := buttonImg.Bounds()
	x := float64(config.SeedChooserButtonX) - float64(buttonBounds.Dx())/2
	y := float64(config.SeedChooserButtonY) - float64(buttonBounds.Dy())/2

	// 绘制按钮
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(buttonImg, op)

	// 如果悬停，绘制发光效果（在按钮之后绘制，覆盖在按钮上方）
	isHoveringButton := s.inputSystem != nil && s.inputSystem.IsHoveringButton()
	if isHoveringButton && s.buttonGlow != nil {
		glowOp := &ebiten.DrawImageOptions{}
		glowBounds := s.buttonGlow.Bounds()
		glowX := float64(config.SeedChooserButtonX) - float64(glowBounds.Dx())/2
		glowY := float64(config.SeedChooserButtonY) - float64(glowBounds.Dy())/2
		glowOp.GeoM.Translate(glowX, glowY)
		screen.DrawImage(s.buttonGlow, glowOp)
	}

	// 绘制按钮文字"一起摇滚吧!"
	if s.titleFont != nil {
		buttonText := "一起摇滚吧!"
		textWidth := text.Advance(buttonText, s.titleFont)
		textX := float64(config.SeedChooserButtonX) - textWidth/2
		// 垂直居中：使用字体大小计算文字高度，居中于按钮
		textHeight := s.titleFont.Size
		textY := float64(config.SeedChooserButtonY) - textHeight/2

		// 阴影
		shadowOp := &text.DrawOptions{}
		shadowOp.GeoM.Translate(textX+1, textY+1)
		shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 180})
		text.Draw(screen, buttonText, s.titleFont, shadowOp)

		// 文字颜色：激活时为金黄色（与标题一致），禁用时为黑色
		textOp := &text.DrawOptions{}
		textOp.GeoM.Translate(textX, textY)
		if isFull {
			textOp.ColorScale.ScaleWithColor(color.RGBA{255, 220, 100, 255}) // 金黄色，与标题一致
		} else {
			textOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 255}) // 黑色
		}
		text.Draw(screen, buttonText, s.titleFont, textOp)
	}
}

// isPlantSelected 检查植物是否已被选中
func (s *SeedChooserRenderSystem) isPlantSelected(plantID string, selectedPlants []string) bool {
	for _, p := range selectedPlants {
		if p == plantID {
			return true
		}
	}
	return false
}

// GetInventoryCardBounds 获取仓库卡片的边界（用于点击检测）
// 返回卡片索引对应的边界矩形
func (s *SeedChooserRenderSystem) GetInventoryCardBounds(index int) (x, y, width, height float64) {
	if index < 0 || index >= len(s.availablePlants) {
		return 0, 0, 0, 0
	}

	row := index / config.SeedInventoryCardsPerRow
	col := index % config.SeedInventoryCardsPerRow
	x = float64(config.SeedInventoryStartX) + float64(col)*(config.SeedInventoryCardWidth+config.SeedInventoryCardSpacingX)
	y = float64(config.SeedInventoryStartY) + float64(row)*(config.SeedInventoryCardHeight+config.SeedInventoryCardSpacingY)
	return x, y, config.SeedInventoryCardWidth, config.SeedInventoryCardHeight
}

// GetSlotBounds 获取卡槽的边界（用于点击检测）
func (s *SeedChooserRenderSystem) GetSlotBounds(index int) (x, y, width, height float64) {
	if index < 0 || index >= config.SeedSlotCount {
		return 0, 0, 0, 0
	}

	x = float64(config.SeedSlotStartX) + float64(index)*(config.SeedSlotWidth+config.SeedSlotSpacing)
	y = float64(config.SeedSlotStartY)
	return x, y, config.SeedSlotWidth, config.SeedSlotHeight
}

// GetButtonBounds 获取按钮的边界（用于点击检测）
func (s *SeedChooserRenderSystem) GetButtonBounds() (x, y, width, height float64) {
	x = float64(config.SeedChooserButtonX) - config.SeedChooserButtonWidth/2
	y = float64(config.SeedChooserButtonY) - config.SeedChooserButtonHeight/2
	return x, y, config.SeedChooserButtonWidth, config.SeedChooserButtonHeight
}

// GetAvailablePlants 获取可用植物列表
func (s *SeedChooserRenderSystem) GetAvailablePlants() []string {
	return s.availablePlants
}

// SetInputSystem 设置输入系统引用（依赖注入）
func (s *SeedChooserRenderSystem) SetInputSystem(is *SeedChooserInputSystem) {
	s.inputSystem = is
}

// DrawFlyingCards 渲染飞行中的卡片
// Story 8.12 重构: 完全复用 entities.RenderPlantCard() 统一渲染逻辑
func (s *SeedChooserRenderSystem) DrawFlyingCards(screen *ebiten.Image) {
	if s.inputSystem == nil {
		return
	}

	flyingCards := s.inputSystem.GetFlyingCards()
	for _, card := range flyingCards {
		if !card.IsFlying {
			continue
		}

		// 获取飞行位置
		x, y := GetFlyingCardPosition(card)

		// 获取植物卡片组件
		plantCard, ok := s.plantCards[card.PlantID]
		if !ok {
			continue
		}

		// 使用统一渲染逻辑
		entities.RenderPlantCard(screen, plantCard, x, y, s.sunFontSource, s.sunFontSize)
	}
}

// loadPlantTooltips 加载植物名称和描述
func (s *SeedChooserRenderSystem) loadPlantTooltips() {
	// 植物ID到LawnStrings键名的映射
	plantKeyMap := map[string]string{
		"sunflower":  "SUNFLOWER",
		"peashooter": "PEASHOOTER",
		"wallnut":    "WALL_NUT",
		"cherrybomb": "CHERRY_BOMB",
		"potatomine": "POTATO_MINE",
		"snowpea":    "SNOW_PEA",
		"chomper":    "CHOMPER",
		"repeater":   "REPEATER",
	}

	// 获取 LawnStrings 实例
	lawnStrings := s.gameState.LawnStrings
	if lawnStrings == nil {
		log.Printf("[SeedChooserRenderSystem] 警告: LawnStrings 未加载，Tooltip 数据不可用")
		return
	}

	// 为每个植物加载名称和描述
	for plantID, keyName := range plantKeyMap {
		// 植物名称
		name := lawnStrings.GetString(keyName)
		s.plantNames[plantID] = name

		// 植物描述（Tooltip）
		tooltipKey := keyName + "_TOOLTIP"
		tooltip := lawnStrings.GetString(tooltipKey)
		// 如果找到了 Tooltip（没有被返回为 [KEY] 格式）
		if tooltip != "["+tooltipKey+"]" {
			s.plantTooltips[plantID] = tooltip
		}
	}

	log.Printf("[SeedChooserRenderSystem] 加载了 %d 个植物的 Tooltip 数据", len(s.plantTooltips))
}

// loadZombieNames 加载僵尸名称
func (s *SeedChooserRenderSystem) loadZombieNames() {
	// 僵尸UnitID到LawnStrings键名的映射
	zombieKeyMap := map[string]string{
		"zombie":           "ZOMBIE",
		"zombie_conehead":  "CONEHEAD_ZOMBIE",
		"zombie_buckethead": "BUCKETHEAD_ZOMBIE",
		"zombie_polevaulter": "POLE_VAULTING_ZOMBIE",
		"zombie_flag":      "FLAG_ZOMBIE",
	}

	// 获取 LawnStrings 实例
	lawnStrings := s.gameState.LawnStrings
	if lawnStrings == nil {
		log.Printf("[SeedChooserRenderSystem] 警告: LawnStrings 未加载，僵尸名称数据不可用")
		return
	}

	// 为每个僵尸加载名称
	for unitID, keyName := range zombieKeyMap {
		name := lawnStrings.GetString(keyName)
		// 如果找到了名称（没有被返回为 [KEY] 格式）
		if name != "["+keyName+"]" {
			s.zombieNames[unitID] = name
		}
	}

	log.Printf("[SeedChooserRenderSystem] 加载了 %d 个僵尸的名称数据", len(s.zombieNames))
}

// DrawTooltip 渲染悬停提示框
// 在选卡界面渲染之后调用
func (s *SeedChooserRenderSystem) DrawTooltip(screen *ebiten.Image) {
	if s.inputSystem == nil {
		return
	}

	// 获取悬停的植物ID
	hoveredPlantID := s.inputSystem.GetHoveredPlantID()

	// 获取悬停的僵尸类型
	hoveredZombieType := s.inputSystem.GetHoveredZombieType()

	// 如果都没有悬停，返回
	if hoveredPlantID == "" && hoveredZombieType == "" {
		return
	}

	// 获取鼠标位置
	mouseX, mouseY := s.inputSystem.GetHoveredMousePosition()

	// 确定要显示的内容
	var displayName, displayTooltip string
	var hasName, hasTooltip bool

	if hoveredPlantID != "" {
		// 植物 Tooltip
		displayName, hasName = s.plantNames[hoveredPlantID]
		displayTooltip, hasTooltip = s.plantTooltips[hoveredPlantID]
	} else if hoveredZombieType != "" {
		// 僵尸 Tooltip（只显示名称）
		displayName, hasName = s.zombieNames[hoveredZombieType]
		hasTooltip = false
	}

	if !hasName && !hasTooltip {
		return
	}

	// 计算 Tooltip 位置（鼠标右下方）
	tooltipX := float64(mouseX) + config.TooltipOffsetX
	tooltipY := float64(mouseY) + config.TooltipOffsetY

	// 计算文本宽度和高度
	var maxWidth float64
	lineCount := 0

	if s.sunFont != nil {
		if hasName {
			nameWidth := text.Advance(displayName, s.sunFont)
			if nameWidth > maxWidth {
				maxWidth = nameWidth
			}
			lineCount++
		}
		if hasTooltip {
			tooltipWidth := text.Advance(displayTooltip, s.sunFont)
			if tooltipWidth > maxWidth {
				maxWidth = tooltipWidth
			}
			lineCount++
		}
	}

	// 计算 Tooltip 尺寸
	tooltipWidth := maxWidth + config.TooltipPadding*2
	tooltipHeight := float64(lineCount)*16 + config.TooltipPadding*2 + float64(lineCount-1)*config.TooltipLineSpacing

	// 确保 Tooltip 不超出屏幕右边界
	screenWidth := 800.0
	if tooltipX+tooltipWidth > screenWidth {
		tooltipX = screenWidth - tooltipWidth - 5
	}

	// 绘制背景（浅黄色）
	bgColor := color.RGBA{255, 255, 204, 255} // #FFFFCC
	s.drawRect(screen, tooltipX, tooltipY, tooltipWidth, tooltipHeight, bgColor)

	// 绘制边框（黑色）
	borderColor := color.RGBA{0, 0, 0, 255}
	s.drawRectBorder(screen, tooltipX, tooltipY, tooltipWidth, tooltipHeight, borderColor, 1)

	// 绘制文本
	if s.sunFont != nil {
		textX := tooltipX + config.TooltipPadding
		textY := tooltipY + config.TooltipPadding

		// 绘制名称（黑色）
		if hasName {
			nameOp := &text.DrawOptions{}
			nameOp.GeoM.Translate(textX, textY)
			nameOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 255})
			text.Draw(screen, displayName, s.sunFont, nameOp)
			textY += 16 + config.TooltipLineSpacing
		}

		// 绘制描述（深灰色）
		if hasTooltip {
			tooltipOp := &text.DrawOptions{}
			tooltipOp.GeoM.Translate(textX, textY)
			tooltipOp.ColorScale.ScaleWithColor(color.RGBA{80, 80, 80, 255})
			text.Draw(screen, displayTooltip, s.sunFont, tooltipOp)
		}
	}
}

// drawRect 绘制填充矩形
func (s *SeedChooserRenderSystem) drawRect(screen *ebiten.Image, x, y, width, height float64, c color.Color) {
	img := ebiten.NewImage(int(width), int(height))
	img.Fill(c)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}

// drawRectBorder 绘制矩形边框
func (s *SeedChooserRenderSystem) drawRectBorder(screen *ebiten.Image, x, y, width, height float64, c color.Color, thickness float64) {
	// 上边
	s.drawRect(screen, x, y, width, thickness, c)
	// 下边
	s.drawRect(screen, x, y+height-thickness, width, thickness, c)
	// 左边
	s.drawRect(screen, x, y, thickness, height, c)
	// 右边
	s.drawRect(screen, x+width-thickness, y, thickness, height, c)
}
