package scenes

// ============================================================================
// 教程系统相关方法
// Story 15.5: 从 game_scene.go 拆分出教程系统逻辑
// ============================================================================

// ============================================================================
// Story 19.3: GuidedTutorialStateProvider 接口实现
// ============================================================================

// IsGuidedTutorialActive 返回强引导模式是否激活
// 实现 systems.GuidedTutorialStateProvider 接口
func (s *GameScene) IsGuidedTutorialActive() bool {
	if s.guidedTutorialSystem == nil {
		return false
	}
	return s.guidedTutorialSystem.IsActive()
}

// IsOperationAllowed 检查操作是否在强引导白名单中
// 实现 systems.GuidedTutorialStateProvider 接口
func (s *GameScene) IsOperationAllowed(operation string) bool {
	if s.guidedTutorialSystem == nil {
		return true // 系统不存在，允许所有操作
	}
	return s.guidedTutorialSystem.IsOperationAllowed(operation)
}

// NotifyOperation 通知强引导系统发生了某个操作
// 实现 systems.GuidedTutorialStateProvider 接口
func (s *GameScene) NotifyOperation(operation string) {
	if s.guidedTutorialSystem == nil {
		return
	}
	s.guidedTutorialSystem.NotifyOperation(operation)
}
