# 暂停菜单验证程序

快速验证和调试暂停菜单UI的独立测试工具。

## 功能特性

- ✅ 完整显示暂停菜单UI（墓碑背景、按钮、滑动条、复选框）
- ✅ 实时调整UI元素位置（方便调试）
- ✅ 显示详细的调试信息
- ✅ 独立运行，不依赖完整游戏流程

## 编译和运行

```bash
# 编译验证程序
go build -o verify_pause_menu cmd/verify_pause_menu/main.go

# 运行（默认模式）
./verify_pause_menu

# 运行（详细模式）
./verify_pause_menu --verbose
```

## 快捷键

| 按键 | 功能 |
|------|------|
| `ESC` | 切换暂停菜单显示/隐藏 |
| `1` | 音乐滑动条Y坐标 -5 |
| `2` | 音乐滑动条Y坐标 +5 |
| `3` | 音效滑动条Y坐标 -5 |
| `4` | 音效滑动条Y坐标 +5 |
| `Q` | 退出程序 |

## UI元素说明

### 1. 墓碑背景
- 资源：`IMAGE_OPTIONS_MENUBACK`
- 位置：屏幕居中
- 透明边缘自动支持

### 2. 按钮
- **返回游戏**：使用原版图片（`options_backtogamebutton0/2`），位于墓碑下方
- **重新开始**：三段式按钮，位于墓碑内部上方
- **主菜单**：三段式按钮，位于墓碑内部中间

### 3. 滑动条
- **音乐滑动条**：第1行，控制背景音乐音量
- **音效滑动条**：第2行，控制音效音量
- 使用资源：`options_sliderslot.png` (滑槽), `options_sliderknob2.png` (滑块)

### 4. 复选框
- **3D加速**：第3行
- **全屏**：第4行
- 使用资源：`options_checkbox0.png` (未选中), `options_checkbox1.png` (已选中)

## 调试技巧

### 调整UI元素位置

1. 运行验证程序
2. 使用数字键 1-4 调整滑动条位置
3. 记录理想的偏移值
4. 在 `pkg/config/layout_config.go` 中更新配置常量：
   ```go
   PauseMenuMusicSliderOffsetY = -120.0  // 调整此值
   PauseMenuSoundSliderOffsetY = -80.0   // 调整此值
   ```

### 查看详细日志

```bash
./verify_pause_menu --verbose
```

将显示：
- 组件创建日志
- 按钮点击回调
- UI元素加载状态

## 预期效果

运行后应看到：
1. ✅ 草坪背景
2. ✅ 半透明黑色遮罩
3. ✅ 居中的墓碑背景
4. ✅ 墓碑下方的"返回游戏"按钮
5. ✅ 墓碑内部的"重新开始"和"主菜单"按钮
6. ✅ 音乐和音效滑动条（带滑块）
7. ✅ 3D加速和全屏复选框

## 故障排查

### 问题：UI元素不显示

**原因**：资源未加载
**解决**：
```bash
# 检查资源文件是否存在
ls assets/images/options_*.png

# 查看详细日志
./verify_pause_menu --verbose | grep "IMAGE_OPTIONS"
```

### 问题：按钮位置错误

**原因**：配置偏移不正确
**解决**：使用快捷键 1-4 实时调整，找到理想值后更新配置文件

### 问题：文字不显示

**原因**：字体文件缺失
**解决**：
```bash
# 检查字体文件
ls assets/fonts/SimHei.ttf
```

## 相关文件

- **主程序**：`cmd/verify_pause_menu/main.go`
- **暂停菜单模块**：`pkg/modules/pause_menu_module.go`
- **配置文件**：`pkg/config/layout_config.go`
- **UI组件**：
  - `pkg/components/slider_component.go`
  - `pkg/components/checkbox_component.go`
  - `pkg/components/button_component.go`

## Story 10.1 相关

此验证程序用于快速检查 Story 10.1 的实现成果：
- ✅ 原版墓碑背景
- ✅ 三个功能按钮（返回游戏/重新开始/主菜单）
- ✅ 音乐/音效滑动条
- ✅ 3D加速/全屏复选框

