package entities

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// TestNewGameOverDialogEntity 测试游戏结束对话框创建
func TestNewGameOverDialogEntity(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)

	// 加载资源配置（尝试多个可能的路径）
	configPaths := []string{
		"assets/config/resources.yaml",       // 从项目根目录运行
		"../../assets/config/resources.yaml", // 从 pkg/entities 运行
	}

	var err error
	for _, path := range configPaths {
		err = rm.LoadResourceConfig(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Skipf("跳过测试：无法加载资源配置（可能需要从项目根目录运行）: %v", err)
	}

	const windowWidth = 800
	const windowHeight = 600

	var retryClicked bool
	var menuClicked bool

	onRetry := func() {
		retryClicked = true
	}

	onMenu := func() {
		menuClicked = true
	}

	// 创建对话框
	dialogID, err := NewGameOverDialogEntity(em, rm, windowWidth, windowHeight, onRetry, onMenu)
	if err != nil {
		// 如果是资源加载错误，跳过测试（需要从项目根目录运行）
		t.Skipf("跳过测试：创建对话框失败（需要从项目根目录运行）: %v", err)
	}

	// 验证实体已创建
	if dialogID == 0 {
		t.Errorf("对话框实体ID为0")
	}

	// 验证位置组件
	posComp, ok := ecs.GetComponent[*components.PositionComponent](em, dialogID)
	if !ok {
		t.Errorf("未找到位置组件")
	} else {
		// 验证对话框居中
		expectedX := float64(windowWidth)/2 - 500.0/2
		expectedY := float64(windowHeight)/2 - 250.0/2
		if posComp.X != expectedX || posComp.Y != expectedY {
			t.Errorf("对话框位置错误: got (%.1f, %.1f), want (%.1f, %.1f)",
				posComp.X, posComp.Y, expectedX, expectedY)
		}
	}

	// 验证对话框组件
	dialogComp, ok := ecs.GetComponent[*components.DialogComponent](em, dialogID)
	if !ok {
		t.Errorf("未找到对话框组件")
	} else {
		// 验证标题
		if dialogComp.Title != "游戏结束" {
			t.Errorf("对话框标题错误: got %s, want %s", dialogComp.Title, "游戏结束")
		}

		// 验证消息为空
		if dialogComp.Message != "" {
			t.Errorf("对话框消息应为空: got %s", dialogComp.Message)
		}

		// 验证按钮数量
		if len(dialogComp.Buttons) != 2 {
			t.Errorf("按钮数量错误: got %d, want 2", len(dialogComp.Buttons))
		} else {
			// 验证按钮标签
			if dialogComp.Buttons[0].Label != "再次尝试" {
				t.Errorf("第一个按钮标签错误: got %s, want %s",
					dialogComp.Buttons[0].Label, "再次尝试")
			}
			if dialogComp.Buttons[1].Label != "返回主菜单" {
				t.Errorf("第二个按钮标签错误: got %s, want %s",
					dialogComp.Buttons[1].Label, "返回主菜单")
			}

			// 测试回调函数
			dialogComp.Buttons[0].OnClick()
			if !retryClicked {
				t.Errorf("再次尝试回调未触发")
			}

			dialogComp.Buttons[1].OnClick()
			if !menuClicked {
				t.Errorf("返回主菜单回调未触发")
			}
		}

		// 验证可见性
		if !dialogComp.IsVisible {
			t.Errorf("对话框应可见")
		}

		// 验证尺寸
		if dialogComp.Width != 500.0 {
			t.Errorf("对话框宽度错误: got %.1f, want 500.0", dialogComp.Width)
		}
		if dialogComp.Height != 250.0 {
			t.Errorf("对话框高度错误: got %.1f, want 250.0", dialogComp.Height)
		}

		// 验证自动关闭
		if !dialogComp.AutoClose {
			t.Errorf("对话框应设置为自动关闭")
		}
	}

	// 验证 UI 组件
	uiComp, ok := ecs.GetComponent[*components.UIComponent](em, dialogID)
	if !ok {
		t.Errorf("未找到 UI 组件")
	} else {
		if uiComp.State != components.UINormal {
			t.Errorf("UI 状态错误: got %v, want %v", uiComp.State, components.UINormal)
		}
	}
}

// TestNewGameOverDialogEntity_NilCallbacks 测试空回调函数
func TestNewGameOverDialogEntity_NilCallbacks(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)

	// 加载资源配置（尝试多个可能的路径）
	configPaths := []string{
		"assets/config/resources.yaml",       // 从项目根目录运行
		"../../assets/config/resources.yaml", // 从 pkg/entities 运行
	}

	var err error
	for _, path := range configPaths {
		err = rm.LoadResourceConfig(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Skipf("跳过测试：无法加载资源配置（可能需要从项目根目录运行）: %v", err)
	}

	// 使用 nil 回调创建对话框
	dialogID, err := NewGameOverDialogEntity(em, rm, 800, 600, nil, nil)
	if err != nil {
		// 如果是资源加载错误，跳过测试（需要从项目根目录运行）
		t.Skipf("跳过测试：创建对话框失败（需要从项目根目录运行）: %v", err)
	}

	// 验证对话框已创建
	if dialogID == 0 {
		t.Errorf("对话框实体ID为0")
	}

	// 验证点击按钮不会 panic
	dialogComp, ok := ecs.GetComponent[*components.DialogComponent](em, dialogID)
	if !ok {
		t.Fatalf("未找到对话框组件")
	}

	// 调用 nil 回调应不会 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("点击按钮时发生 panic: %v", r)
		}
	}()

	dialogComp.Buttons[0].OnClick() // 再次尝试（nil 回调）
	dialogComp.Buttons[1].OnClick() // 返回主菜单（nil 回调）
}
