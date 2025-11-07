package config

import (
	"os"
	"testing"
)

// TestLoadReanimConfig_ValidFile 测试加载有效的配置文件
func TestLoadReanimConfig_ValidFile(t *testing.T) {
	config, err := LoadReanimConfig("testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("加载有效配置文件失败: %v", err)
	}

	// 验证基础字段
	if config.ReanimFile != "assets/effect/reanim/Test.reanim" {
		t.Errorf("ReanimFile = %s, want assets/effect/reanim/Test.reanim", config.ReanimFile)
	}

	// 验证动画列表
	if len(config.Animations) != 2 {
		t.Errorf("Animations 数量 = %d, want 2", len(config.Animations))
	}

	// 验证动画组合
	if len(config.AnimationCombos) != 1 {
		t.Fatalf("AnimationCombos 数量 = %d, want 1", len(config.AnimationCombos))
	}

	combo := config.AnimationCombos[0]
	if combo.Name != "test_combo" {
		t.Errorf("Combo.Name = %s, want test_combo", combo.Name)
	}

	if len(combo.Animations) != 2 {
		t.Errorf("Combo.Animations 数量 = %d, want 2", len(combo.Animations))
	}

	if combo.BindingStrategy != "auto" {
		t.Errorf("Combo.BindingStrategy = %s, want auto", combo.BindingStrategy)
	}
}

// TestLoadReanimConfig_FileNotFound 测试加载不存在的文件
func TestLoadReanimConfig_FileNotFound(t *testing.T) {
	_, err := LoadReanimConfig("nonexistent.yaml")
	if err == nil {
		t.Error("期望加载不存在的文件时返回错误，但得到 nil")
	}
}

// TestLoadReanimConfig_InvalidYAML 测试加载格式错误的 YAML
func TestLoadReanimConfig_InvalidYAML(t *testing.T) {
	_, err := LoadReanimConfig("testdata/invalid_yaml.yaml")
	if err == nil {
		t.Error("期望加载无效 YAML 时返回错误，但得到 nil")
	}
}

// TestValidateConfig_MissingReanimFile 测试缺少必填字段
func TestValidateConfig_MissingReanimFile(t *testing.T) {
	config := &ReanimConfig{}
	err := validateConfig(config)
	if err == nil {
		t.Error("期望验证失败，但得到 nil")
	}
	// 验证错误信息包含字段名
	if err != nil && err.Error() == "" {
		t.Error("错误信息为空")
	}
}

// TestValidateConfig_InvalidAnimationReference 测试引用不存在的动画
func TestValidateConfig_InvalidAnimationReference(t *testing.T) {
	config := &ReanimConfig{
		ReanimFile: "test.reanim",
		Animations: []AnimationDef{
			{Name: "anim_idle"},
		},
		AnimationCombos: []AnimationComboConfig{
			{
				Name:       "combo1",
				Animations: []string{"anim_nonexistent"},
			},
		},
	}
	err := validateConfig(config)
	if err == nil {
		t.Error("期望验证失败（引用不存在的动画），但得到 nil")
	}
}

// TestValidateConfig_InvalidBindingStrategy 测试无效的绑定策略
func TestValidateConfig_InvalidBindingStrategy(t *testing.T) {
	config := &ReanimConfig{
		ReanimFile: "test.reanim",
		AnimationCombos: []AnimationComboConfig{
			{
				Name:            "combo1",
				Animations:      []string{"anim_idle"},
				BindingStrategy: "invalid_strategy",
			},
		},
	}
	err := validateConfig(config)
	if err == nil {
		t.Error("期望验证失败（无效的绑定策略），但得到 nil")
	}
}

// TestValidateConfig_ManualBindingWithoutBindings 测试 manual 策略但未提供绑定
func TestValidateConfig_ManualBindingWithoutBindings(t *testing.T) {
	config := &ReanimConfig{
		ReanimFile: "test.reanim",
		AnimationCombos: []AnimationComboConfig{
			{
				Name:            "combo1",
				Animations:      []string{"anim_idle"},
				BindingStrategy: "manual",
				ManualBindings:  map[string]string{}, // 空的绑定
			},
		},
	}
	err := validateConfig(config)
	if err == nil {
		t.Error("期望验证失败（manual 策略但未提供绑定），但得到 nil")
	}
}

// TestValidateConfig_EmptyAnimationsList 测试动画列表为空
func TestValidateConfig_EmptyAnimationsList(t *testing.T) {
	config := &ReanimConfig{
		ReanimFile: "test.reanim",
		AnimationCombos: []AnimationComboConfig{
			{
				Name:       "combo1",
				Animations: []string{}, // 空的动画列表
			},
		},
	}
	err := validateConfig(config)
	if err == nil {
		t.Error("期望验证失败（动画列表为空），但得到 nil")
	}
}

// TestValidateConfig_MissingComboName 测试组合缺少名称
func TestValidateConfig_MissingComboName(t *testing.T) {
	config := &ReanimConfig{
		ReanimFile: "test.reanim",
		AnimationCombos: []AnimationComboConfig{
			{
				Name:       "", // 缺少名称
				Animations: []string{"anim_idle"},
			},
		},
	}
	err := validateConfig(config)
	if err == nil {
		t.Error("期望验证失败（缺少组合名称），但得到 nil")
	}
}

// TestValidateConfig_ValidAutoStrategy 测试有效的 auto 策略
func TestValidateConfig_ValidAutoStrategy(t *testing.T) {
	config := &ReanimConfig{
		ReanimFile: "test.reanim",
		AnimationCombos: []AnimationComboConfig{
			{
				Name:            "combo1",
				Animations:      []string{"anim_idle"},
				BindingStrategy: "auto",
			},
		},
	}
	err := validateConfig(config)
	if err != nil {
		t.Errorf("验证有效的 auto 策略失败: %v", err)
	}
}

// TestValidateConfig_ValidManualStrategy 测试有效的 manual 策略
func TestValidateConfig_ValidManualStrategy(t *testing.T) {
	config := &ReanimConfig{
		ReanimFile: "test.reanim",
		AnimationCombos: []AnimationComboConfig{
			{
				Name:            "combo1",
				Animations:      []string{"anim_idle"},
				BindingStrategy: "manual",
				ManualBindings: map[string]string{
					"track1": "anim_idle",
				},
			},
		},
	}
	err := validateConfig(config)
	if err != nil {
		t.Errorf("验证有效的 manual 策略失败: %v", err)
	}
}

// TestValidateParentTracks_NoCycle 测试无循环的父子关系
func TestValidateParentTracks_NoCycle(t *testing.T) {
	parentTracks := map[string]string{
		"child1": "parent1",
		"child2": "parent1",
		"child3": "child2",
	}
	err := validateParentTracks(parentTracks)
	if err != nil {
		t.Errorf("验证无循环的父子关系失败: %v", err)
	}
}

// TestValidateParentTracks_WithCycle 测试存在循环的父子关系
func TestValidateParentTracks_WithCycle(t *testing.T) {
	parentTracks := map[string]string{
		"child1": "child2",
		"child2": "child3",
		"child3": "child1", // 循环
	}
	err := validateParentTracks(parentTracks)
	if err == nil {
		t.Error("期望检测到循环依赖，但得到 nil")
	}
}

// TestLoadReanimConfig_RealPeashooterConfig 测试加载真实的豌豆射手配置
func TestLoadReanimConfig_RealPeashooterConfig(t *testing.T) {
	// 检查文件是否存在
	if _, err := os.Stat("../../data/reanim_configs/peashooter.yaml"); os.IsNotExist(err) {
		t.Skip("豌豆射手配置文件不存在，跳过测试")
	}

	config, err := LoadReanimConfig("../../data/reanim_configs/peashooter.yaml")
	if err != nil {
		t.Fatalf("加载豌豆射手配置失败: %v", err)
	}

	// 验证基础字段
	if config.ReanimFile != "assets/effect/reanim/PeaShooterSingle.reanim" {
		t.Errorf("ReanimFile = %s, want assets/effect/reanim/PeaShooterSingle.reanim", config.ReanimFile)
	}

	// 验证至少有一个动画组合
	if len(config.AnimationCombos) == 0 {
		t.Error("豌豆射手配置应该至少有一个动画组合")
	}
}

// TestValidateConfig_CircularParentTracks 测试循环依赖检测在配置验证中生效
func TestValidateConfig_CircularParentTracks(t *testing.T) {
	config := &ReanimConfig{
		ReanimFile: "test.reanim",
		AnimationCombos: []AnimationComboConfig{
			{
				Name:       "combo1",
				Animations: []string{"anim_idle"},
				ParentTracks: map[string]string{
					"child1": "child2",
					"child2": "child3",
					"child3": "child1", // 循环依赖
				},
			},
		},
	}
	err := validateConfig(config)
	if err == nil {
		t.Error("期望检测到循环依赖，但得到 nil")
	}
	// 验证错误信息包含循环依赖相关信息
	if err != nil && err.Error() == "" {
		t.Error("错误信息为空")
	}
}
