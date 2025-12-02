# Sprint 变更提案：「一大波僵尸正在接近!」动画实现修正

## 文档信息

| 项目 | 内容 |
|------|------|
| **提案标题** | 「一大波僵尸正在接近!」动画实现修正 |
| **提案日期** | 2025-12-02 |
| **触发来源** | Story 17.7 实现细节偏差 |
| **提案状态** | ✅ 已批准 |
| **批准日期** | 2025-12-02 |

---

## 1. 问题摘要

### 1.1 问题描述

Story 17.7（旗帜波特殊计时与红字警告）已标记完成，但「一大波僵尸正在接近!」的实现方式与原版不符：

| 方面 | 当前实现 | 原版实现 |
|------|---------|---------|
| **文字来源** | 复用 `FinalWave.png`（显示「最后一波」） | 使用 `HouseofTerror28` 位图字体渲染 |
| **文字内容** | 「最后一波」（4个字） | 「一大波僵尸正在接近!」（10个字） |
| **文字颜色** | 预制红色图片 | 动态着色为红色 |
| **动画定义** | `FinalWave.reanim` | 复用 `FinalWave.reanim` 动画参数 |

### 1.2 根本原因

- Story 17.7 开发时误解了原版机制
- 错误地认为「一大波僵尸正在接近!」和「最后一波」使用相同的预制图片
- 实际上原版使用位图字体动态渲染不同文字

### 1.3 资源文件

原版实现依赖以下资源：
- `assets/data/HouseofTerror28.txt` - 位图字体元数据（字符映射、宽度、矩形区域）
- `assets/data/HouseofTerror28.png` - 位图字体图集（包含中文字符）

---

## 2. Epic 影响摘要

| Epic | 影响 | 说明 |
|------|------|------|
| **Epic 17** | 需追加任务 | Story 17.7 需要补充修复任务 |
| **其他 Epic** | 无影响 | 这是独立的 UI 实现修复 |

---

## 3. 工件调整需求

### 3.1 需要修改的文件

| 文件路径 | 变更类型 | 变更描述 |
|---------|---------|---------|
| `pkg/utils/bitmap_font.go` | **扩展** | 添加 `RenderTextToImage(text, color)` 方法 |
| `pkg/systems/flag_wave_warning_system.go` | **重构** | 使用 `BitmapFont` 动态渲染文字图片 |
| `pkg/utils/bitmap_font_test.go` | **扩展** | 添加着色渲染测试 |
| `pkg/systems/flag_wave_warning_system_test.go` | **更新** | 更新测试以覆盖新实现 |

### 3.2 不需要修改的文件

| 文件路径 | 原因 |
|---------|------|
| `data/reanim/FinalWave.reanim` | 动画定义复用，无需修改 |
| `data/reanim_config/finalwave.yaml` | 配置复用，无需修改 |
| `assets/reanim/FinalWave.png` | 「最后一波」专用，保持不变 |

---

## 4. 推荐前进路径

### 4.1 方案选择

**选择方案**：直接调整/集成

**选择理由**：
1. 现有代码结构合理，只需局部修改
2. `BitmapFont` 工具类已存在，扩展简单
3. 不影响已完成的其他功能
4. 工作量最小，风险最低

### 4.2 技术方案

```
┌─────────────────────────────────────────────────────────────┐
│                    修改前架构                                │
├─────────────────────────────────────────────────────────────┤
│  FlagWaveWarningSystem                                      │
│       │                                                     │
│       ├── 加载 FinalWave.reanim                             │
│       ├── 绑定 IMAGE_REANIM_FINALWAVE (FinalWave.png)       │
│       └── 播放动画 → 显示「最后一波」❌                       │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    修改后架构                                │
├─────────────────────────────────────────────────────────────┤
│  FlagWaveWarningSystem                                      │
│       │                                                     │
│       ├── 加载 HouseofTerror28 位图字体                      │
│       ├── 渲染「一大波僵尸正在接近!」→ 红色图片               │
│       ├── 加载 FinalWave.reanim (复用动画参数)              │
│       ├── 替换轨道图片为动态生成的红色文字                    │
│       └── 播放动画 → 显示「一大波僵尸正在接近!」✅            │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. PRD MVP 影响

| 检查项 | 状态 |
|--------|------|
| 原始 MVP 仍可实现？ | ✅ 是 |
| MVP 范围需要缩减？ | ❌ 否 |
| 核心 MVP 目标需要修改？ | ❌ 否 |

**结论**：这是实现细节修正，不影响 MVP 范围和目标。

---

## 6. 高层行动计划

### 6.1 任务分解

| # | 任务 | 描述 | 估计时间 |
|---|------|------|---------|
| 1 | **扩展 BitmapFont** | 添加 `RenderTextToImage(text string, tintColor color.Color) *ebiten.Image` 方法，支持将文字渲染为带颜色的图像 | 1-2 小时 |
| 2 | **重构 FlagWaveWarningSystem** | 修改 `createWarningEntity()` 方法，使用 `BitmapFont` 动态生成红色文字图片，替代固定的 `FinalWave.png` | 2-3 小时 |
| 3 | **添加单元测试** | 为 `BitmapFont.RenderTextToImage()` 添加测试；更新 `FlagWaveWarningSystem` 测试 | 1-2 小时 |
| 4 | **集成验证** | 运行游戏验证动画效果，确保文字、颜色、动画参数正确 | 1 小时 |

**总估计时间**：5-8 小时

### 6.2 实施顺序

```
Task 1 (BitmapFont 扩展)
    │
    ▼
Task 2 (FlagWaveWarningSystem 重构)
    │
    ▼
Task 3 (单元测试)
    │
    ▼
Task 4 (集成验证)
```

---

## 7. Agent 交接计划

| 角色 | 职责 | 说明 |
|------|------|------|
| **SM** | 完成变更提案 | 本文档 ✅ |
| **Dev Agent** | 实施代码修改 | Task 1-4 |
| **QA Agent** | 验证修复 | 可选：更新 Gate 文件 |

**不需要**：
- ❌ PM 介入（非范围变更）
- ❌ Architect 介入（架构无变化）

---

## 8. 变更记录

| 日期 | 版本 | 描述 | 作者 |
|------|------|------|------|
| 2025-12-02 | 1.0 | 初始提案创建并批准 | Bob (Scrum Master) |

---

## 附录：相关文件引用

- Story 文档: `docs/stories/17.7.flag-wave-timing.story.md`
- PRD 章节: `docs/prd/epic-17-zombie-generation-engine.md` (Story 17.7)
- 位图字体工具: `pkg/utils/bitmap_font.go`
- 警告系统: `pkg/systems/flag_wave_warning_system.go`
- 字体资源: `assets/data/HouseofTerror28.txt`, `assets/data/HouseofTerror28.png`
