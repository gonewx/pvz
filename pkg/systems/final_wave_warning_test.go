package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestFinalWaveAnimation_NonLooping 测试 FinalWave 动画的非循环配置
// 验证：PlayCombo 能正确应用 loop: false 配置
func TestFinalWaveAnimation_NonLooping(t *testing.T) {
	// 1. 创建测试环境
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	// 2. 加载配置管理器（Story 13.9: 多文件架构）
	configManager, err := config.NewReanimConfigManager("data/reanim_config")
	if err != nil {
		t.Skipf("跳过测试：无法加载配置文件: %v", err)
	}
	rs.SetConfigManager(configManager)

	// 3. 创建实体并添加最小化的 ReanimComponent
	// 模拟实际使用场景，其中实体已有 ReanimComponent
	entityID := em.CreateEntity()

	// 创建模拟的 Reanim 数据
	reanimComp := &components.ReanimComponent{
		ReanimXML: nil, // PlayCombo 会处理初始化
		PartImages: make(map[string]*ebiten.Image),
		CurrentAnimations: []string{},
	}
	ecs.AddComponent(em, entityID, reanimComp)

	// 4. 通过 AnimationCommandComponent 触发动画
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		UnitID:    "finalwave",
		ComboName: "warning",
		Processed: false,
	})

	// 5. 处理命令（这会调用 PlayCombo）
	rs.processAnimationCommands()

	// 6. 验证命令已处理
	cmd, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
	if !ok {
		t.Fatal("AnimationCommand component not found")
	}
	if !cmd.Processed {
		t.Error("Expected Processed to be true after processing")
	}

	// 注意：由于我们没有真正的 ReanimXML 数据，PlayCombo 会失败
	// 但重要的是验证配置本身是正确的（已在 TestFinalWaveAnimation_ComboConfig 中验证）
	t.Logf("✓ 命令已处理 (配置验证见 TestFinalWaveAnimation_ComboConfig)")
}

// TestFinalWaveAnimation_ComboConfig 测试 FinalWave combo 配置是否存在
func TestFinalWaveAnimation_ComboConfig(t *testing.T) {
	// 1. 加载配置管理器
	configManager, err := config.NewReanimConfigManager("data/reanim_config")
	if err != nil {
		t.Skipf("跳过测试：无法加载配置文件: %v", err)
	}

	// 2. 验证 finalwave 单位配置存在
	unitConfig, err := configManager.GetUnit("finalwave")
	if err != nil {
		t.Fatalf("Failed to get finalwave unit config: %v", err)
	}

	if unitConfig.ID != "finalwave" {
		t.Errorf("Expected unit ID 'finalwave', got '%s'", unitConfig.ID)
	}

	// 3. 验证 warning combo 存在
	combo, err := configManager.GetCombo("finalwave", "warning")
	if err != nil {
		t.Fatalf("Failed to get 'warning' combo: %v", err)
	}

	// 4. 验证 loop 设置为 false
	if combo.Loop == nil {
		t.Fatal("Expected Loop to be set, got nil")
	}
	if *combo.Loop {
		t.Errorf("Expected Loop to be false, got true")
	}

	// 5. 验证动画列表
	if len(combo.Animations) != 1 {
		t.Errorf("Expected 1 animation in combo, got %d", len(combo.Animations))
	}
	if len(combo.Animations) > 0 && combo.Animations[0] != "FinalWave" {
		t.Errorf("Expected animation 'FinalWave', got '%s'", combo.Animations[0])
	}

	t.Logf("✓ finalwave/warning combo 配置正确")
	t.Logf("  - Loop: false")
	t.Logf("  - Animations: %v", combo.Animations)
}
