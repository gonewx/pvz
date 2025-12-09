package entities

import (
	"fmt"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
)

// NewCrazyDaveEntity 创建疯狂戴夫实体
// Dave 是游戏中的教学角色，会在关卡中出现并与玩家对话
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载 Dave 的 Reanim 资源）
//   - dialogueKeys: 对话文本 key 列表（从 LawnStrings.txt 加载）
//   - onComplete: 对话完成回调（所有对话结束且 Dave 离场后调用）
//
// 返回:
//   - ecs.EntityID: 创建的 Dave 实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
//
// 注意：
//   - Dave 创建时处于 Entering 状态，会自动播放 anim_enter 入场动画
//   - 入场动画完成后切换到 Talking 状态，显示对话气泡
//   - 动画和状态转换由 DaveDialogueSystem 处理
func NewCrazyDaveEntity(
	em *ecs.EntityManager,
	rm ResourceLoader,
	dialogueKeys []string,
	onComplete func(),
) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}
	if len(dialogueKeys) == 0 {
		return 0, fmt.Errorf("dialogue keys cannot be empty")
	}

	// 位置由动画控制，初始设为 (0, 0)
	// 动画文件 CrazyDave.reanim 中 anim_enter 定义了入场轨迹：
	// - Dave_body1 轨道 X 从 -356.9 移动到 -55.9
	// 因此不需要手动指定起始位置
	startX := config.DaveTargetX // 0.0
	posY := config.DaveTargetY   // 0.0

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（初始位置在屏幕左侧外）
	em.AddComponent(entityID, &components.PositionComponent{
		X: startX,
		Y: posY,
	})

	// 从 ResourceManager 获取 CrazyDave 的 Reanim 数据和部件图片
	reanimXML := rm.GetReanimXML("CrazyDave")
	partImages := rm.GetReanimPartImages("CrazyDave")

	if reanimXML == nil || partImages == nil {
		return 0, fmt.Errorf("failed to load CrazyDave Reanim resources")
	}

	// 添加 ReanimComponent
	em.AddComponent(entityID, &components.ReanimComponent{
		ReanimName: "CrazyDave",
		ReanimXML:  reanimXML,
		PartImages: partImages,
		IsLooping:  false, // 入场动画不循环，播放一次后由系统切换到 idle
		IsPaused:   false,
		ScaleX:     1.0,
		ScaleY:     1.0,
	})

	// 添加 UIComponent（Dave 作为 UI 元素，不受摄像机影响）
	em.AddComponent(entityID, &components.UIComponent{})

	// 添加 AnimationCommand 组件，播放入场动画
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		UnitID:    "crazydave",
		ComboName: "anim_enter",
		Processed: false,
	})

	// 添加 DaveDialogueComponent
	ecs.AddComponent(em, entityID, &components.DaveDialogueComponent{
		DialogueKeys:       dialogueKeys,
		CurrentLineIndex:   0,
		CurrentText:        "",
		CurrentExpressions: nil,
		IsVisible:          false, // 入场动画期间不显示对话气泡
		State:              components.DaveStateEntering,
		Expression:         "",
		BubbleOffsetX:      config.DaveBubbleOffsetX,
		BubbleOffsetY:      config.DaveBubbleOffsetY,
		OnCompleteCallback: onComplete,
	})

	log.Printf("[DaveFactory] Created CrazyDave entity %d with %d dialogue keys, start pos (%.1f, %.1f)",
		entityID, len(dialogueKeys), startX, posY)

	return entityID, nil
}
