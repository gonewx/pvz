# Animation Showcase 快速开始指南

## 基本使用流程

### 1. 启动程序
```bash
# 方式一：使用 go run
go run cmd/animation_showcase/main.go \
       cmd/animation_showcase/animation_cell.go \
       cmd/animation_showcase/config.go \
       cmd/animation_showcase/grid_layout.go

# 方式二：编译后运行
go build -o animation_showcase \
  cmd/animation_showcase/main.go \
  cmd/animation_showcase/animation_cell.go \
  cmd/animation_showcase/config.go \
  cmd/animation_showcase/grid_layout.go

./animation_showcase --config=cmd/animation_showcase/config.yaml
```

### 2. 网格模式 - 浏览动画
启动后默认进入网格模式，显示多个动画单元：

1. **选择单元**: 点击任意单元
2. **切换动画**: 使用 `←/→` 方向键
3. **翻页**: 使用 `PageDown/PageUp`
4. **快速跳转**: 按数字键 `1-9` 跳转到指定页面

### 3. 单个模式 - 详细查看
选中单元后，按 `Enter` 进入单个展示模式：

1. **全屏显示**: 动画在屏幕中央放大显示
2. **切换动画**: 使用 `←/→` 方向键
3. **返回网格**: 按 `Enter` 键

### 4. 轨道编辑 - 调试动画
在单个模式下，可以编辑部件轨道的显示：

1. **查看轨道列表**: 屏幕左侧显示所有部件轨道
2. **切换轨道**: 按 `F1-F12` 切换对应轨道的显示/隐藏
3. **重置轨道**: 按 `R` 键重置所有轨道为可见

## 键盘快捷键速查表

| 按键 | 网格模式 | 单个模式 |
|------|---------|---------|
| Enter | 切换到单个模式 | 返回网格模式 |
| ←/→ | 切换动画 | 切换动画 |
| PageDown/Up | 翻页 | - |
| 1-9 | 跳转页面 | - |
| F1-F12 | - | 切换轨道 |
| R | - | 重置轨道 |
| H | 帮助开关 | 帮助开关 |
| Tab | 帮助位置 | 帮助位置 |
| ESC | 退出 | 退出 |

## 更新日期
2025-11-07
