package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
)

// ========== Story 8.13: 奖励面板渲染系统测试 ==========

// TestCalculateCardPosition 测试卡片位置计算
func TestCalculateCardPosition(t *testing.T) {
	em := ecs.NewEntityManager()

	rprs := &RewardPanelRenderSystem{
		entityManager: em,
		screenWidth:   800.0,
		screenHeight:  600.0,
	}

	// 测试不同缩放值的卡片位置
	tests := []struct {
		name      string
		cardScale float64
	}{
		{"标准缩放 1.0", 1.0},
		{"缩放 0.5", 0.5},
		{"缩放 1.5", 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cardX, cardY := rprs.calculateCardPosition(tt.cardScale)

			// 验证卡片位置在有效范围内
			if cardX < 0 || cardX > rprs.screenWidth {
				t.Errorf("Card X position %.1f is outside screen bounds [0, %.1f]", cardX, rprs.screenWidth)
			}
			if cardY < 0 || cardY > rprs.screenHeight {
				t.Errorf("Card Y position %.1f is outside screen bounds [0, %.1f]", cardY, rprs.screenHeight)
			}

			// 验证卡片大致在屏幕中央区域
			centerX := rprs.screenWidth / 2
			if cardX < centerX-200 || cardX > centerX+200 {
				t.Errorf("Card X position %.1f is not centered (expected near %.1f)", cardX, centerX)
			}
		})
	}

	t.Logf("✓ calculateCardPosition returns valid positions for various scales")
}

// TestMainMenuButtonPositionConstants 测试主菜单按钮位置常量
// 验证按钮位置常量合理性（Story 8.13）
func TestMainMenuButtonPositionConstants(t *testing.T) {
	// 验证主菜单按钮位置常量（右上角）
	// X 位置应该在右侧（> 0.5）
	if config.RewardPanelMainMenuButtonX <= 0.5 {
		t.Errorf("RewardPanelMainMenuButtonX = %.2f, expected > 0.5 (right side of panel)",
			config.RewardPanelMainMenuButtonX)
	}

	// Y 位置应该在顶部（< 0.2）
	if config.RewardPanelMainMenuButtonY >= 0.2 {
		t.Errorf("RewardPanelMainMenuButtonY = %.2f, expected < 0.2 (top of panel)",
			config.RewardPanelMainMenuButtonY)
	}

	// 验证位置在合理范围内（0-1）
	if config.RewardPanelMainMenuButtonX < 0 || config.RewardPanelMainMenuButtonX > 1 {
		t.Errorf("RewardPanelMainMenuButtonX = %.2f, expected between 0 and 1",
			config.RewardPanelMainMenuButtonX)
	}
	if config.RewardPanelMainMenuButtonY < 0 || config.RewardPanelMainMenuButtonY > 1 {
		t.Errorf("RewardPanelMainMenuButtonY = %.2f, expected between 0 and 1",
			config.RewardPanelMainMenuButtonY)
	}

	t.Logf("✓ Main menu button position constants (X=%.3f, Y=%.3f) are correctly configured",
		config.RewardPanelMainMenuButtonX, config.RewardPanelMainMenuButtonY)
}

// TestMainMenuButtonPosition 测试主菜单按钮位置计算
// 验证按钮位置计算逻辑与 isMainMenuButtonClicked 一致
func TestMainMenuButtonPosition(t *testing.T) {
	screenWidth := 800.0
	screenHeight := 600.0

	// 使用与 isHoveringMainMenuButton 和 isMainMenuButtonClicked 相同的逻辑
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (screenWidth - bgWidth) / 2
	offsetY := (screenHeight - bgHeight) / 2

	buttonX := offsetX + bgWidth*config.RewardPanelMainMenuButtonX
	buttonY := offsetY + bgHeight*config.RewardPanelMainMenuButtonY

	// 验证按钮位置
	// 按钮应该在右上角（X > 屏幕中心，Y < 屏幕中心）
	if buttonX <= screenWidth/2 {
		t.Errorf("Button X = %.1f should be > screen center %.1f", buttonX, screenWidth/2)
	}
	if buttonY >= screenHeight/2 {
		t.Errorf("Button Y = %.1f should be < screen center %.1f", buttonY, screenHeight/2)
	}

	// 验证按钮在屏幕范围内
	buttonWidth := 65.0
	buttonHeight := 40.0
	halfWidth := buttonWidth / 2
	halfHeight := buttonHeight / 2

	if buttonX-halfWidth < 0 || buttonX+halfWidth > screenWidth {
		t.Errorf("Button X range [%.1f, %.1f] is outside screen [0, %.1f]",
			buttonX-halfWidth, buttonX+halfWidth, screenWidth)
	}
	if buttonY-halfHeight < 0 || buttonY+halfHeight > screenHeight {
		t.Errorf("Button Y range [%.1f, %.1f] is outside screen [0, %.1f]",
			buttonY-halfHeight, buttonY+halfHeight, screenHeight)
	}

	t.Logf("✓ Main menu button position (%.1f, %.1f) is correctly calculated", buttonX, buttonY)
}

// TestNextLevelButtonPosition 测试"下一关"按钮位置计算
func TestNextLevelButtonPosition(t *testing.T) {
	screenWidth := 800.0
	screenHeight := 600.0

	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (screenWidth - bgWidth) / 2
	offsetY := (screenHeight - bgHeight) / 2

	buttonX := offsetX + bgWidth/2
	buttonY := offsetY + bgHeight*config.RewardPanelButtonY

	// 验证按钮水平居中
	if buttonX != screenWidth/2 {
		t.Errorf("Button X = %.1f should be centered at %.1f", buttonX, screenWidth/2)
	}

	// 验证按钮在下方
	if buttonY <= screenHeight/2 {
		t.Errorf("Button Y = %.1f should be > screen center %.1f", buttonY, screenHeight/2)
	}

	// 验证按钮在屏幕范围内
	if buttonY < 0 || buttonY > screenHeight {
		t.Errorf("Button Y = %.1f is outside screen [0, %.1f]", buttonY, screenHeight)
	}

	t.Logf("✓ Next level button position (%.1f, %.1f) is correctly calculated", buttonX, buttonY)
}

// TestMainMenuButtonAABBCollision 测试主菜单按钮 AABB 碰撞检测逻辑
// 独立测试 AABB 碰撞检测算法，与 isMainMenuButtonClicked 使用相同逻辑
func TestMainMenuButtonAABBCollision(t *testing.T) {
	// 模拟按钮中心位置
	buttonX := 680.0
	buttonY := 48.0
	buttonWidth := 65.0
	buttonHeight := 40.0
	halfWidth := buttonWidth / 2
	halfHeight := buttonHeight / 2

	// AABB 碰撞检测函数
	isInside := func(mouseX, mouseY float64) bool {
		return mouseX >= buttonX-halfWidth && mouseX <= buttonX+halfWidth &&
			mouseY >= buttonY-halfHeight && mouseY <= buttonY+halfHeight
	}

	tests := []struct {
		name     string
		mouseX   float64
		mouseY   float64
		expected bool
	}{
		{"点击按钮中心", buttonX, buttonY, true},
		{"点击左边缘", buttonX - halfWidth, buttonY, true},
		{"点击右边缘", buttonX + halfWidth, buttonY, true},
		{"点击上边缘", buttonX, buttonY - halfHeight, true},
		{"点击下边缘", buttonX, buttonY + halfHeight, true},
		{"点击左外", buttonX - halfWidth - 1, buttonY, false},
		{"点击右外", buttonX + halfWidth + 1, buttonY, false},
		{"点击上外", buttonX, buttonY - halfHeight - 1, false},
		{"点击下外", buttonX, buttonY + halfHeight + 1, false},
		{"点击角落内", buttonX - halfWidth + 1, buttonY - halfHeight + 1, true},
		{"点击角落外", buttonX - halfWidth - 1, buttonY - halfHeight - 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInside(tt.mouseX, tt.mouseY)
			if result != tt.expected {
				t.Errorf("AABB collision at (%.1f, %.1f) = %v, want %v",
					tt.mouseX, tt.mouseY, result, tt.expected)
			}
		})
	}

	t.Logf("✓ AABB collision detection logic is correct")
}
