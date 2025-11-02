# 主菜单界面完整规格说明

> 基于原版《植物大战僵尸》主菜单界面截图分析
>
> 文档版本：v1.0
> 创建日期：2025-11-01
> 截图来源：`.meta/screenshot/menu/`

---

## 🎨 整体布局

**屏幕划分：**
- 左侧区域（约40%）：装饰性背景元素
- 右侧区域（约60%）：主要交互界面（石碑菜单）

---

## 一、背景元素

### 1. 天空背景
- 渐变蓝天（上深下浅）
- 动态白云

### 2. 左侧场景
- **大树**：左上角，棕色树干，绿色树冠
- **骷髅装饰**：挂在树枝上
- **木质告示牌组**（两块木牌）：
  - **上牌**：
    - 绿色文字："欢迎回来，我的朋友！"
    - 微黄的文字："用户名"
  - **下牌**：
    - 白色中文："如果这不是你的存档，请点我！"

### 3. 中间场景
- **戴夫的房子**：
  - 红色/橙色砖瓦屋顶
  - 米黄色墙壁
  - 带烟囱
  - 有围栏和小花园
  - 房屋窗户有暖黄色灯光

### 4. 前景元素
- 草地（绿色）
- 装饰性草丛
- 石头散落
- **点击菜单后，僵尸手掌动画**：从地下升起的动画效果。

---

## 二、主界面石碑

### 外观特征
- **材质**：深灰色石质纹理
- **形状**：墓碑状，顶部不规则边缘
- **雕刻文字**：凹陷效果的中文字体

### 文字布局（从上到下）

#### 1. 顶部标识
- **内容**："LEVEL 1-4"（英文，较小字号）
- **位置**：石碑上端中央
- **功能**：显示当前最新关卡进度

#### 2. 主菜单选项（大号中文）
按从上到下顺序：

| 序号 | 选项名称 | 功能描述 | 解锁条件 |
|------|---------|---------|---------|
| 1 | 冒险模式 | 进入关卡选择/开始游戏 | 默认解锁 |
| 2 | 玩玩小游戏 | 进入小游戏列表 | 通关特定关卡 |
| 3 | 解谜模式 | 进入解谜关卡列表 | 通关特定关卡 |
| 4 | 生存模式 | 进入生存挑战模式 | 需完成更多冒险 |

---

## 三、功能按钮区

**位置**：右下角，石碑下方

### 按钮组（3个花瓶形按钮）

| 按钮名称 | 位置 | 功能 |
|---------|------|------|
| 选项 | 左侧 | 打开游戏设置（音量、画质等） |
| 帮助 | 中间 | 显示帮助文档/教程 |
| 退出 | 右侧 | 退出游戏 |

### 视觉状态

- **正常态**：黑色文字

### 交互反馈
- 鼠标悬停 → 绿色文字
- 点击音效 → `SOUND_BUTTONCLICK`

---

## 四、对话框系统

### 未解锁提示对话框

**触发场景**：点击未解锁的模式（如"生存模式"）

#### 对话框结构

**外观**：
- 紫灰色半透明背景
- 顶部装饰：骷髅头图案
- 边框：不规则石质边缘
- 居中显示，覆盖主界面

**内容布局**：
- **标题**：黄色文字 "未解锁！"
- **说明文字**：黄色文字（根据不同模式显示不同提示）
  - 示例："进行更多新冒险来解锁生存模式。"
- **确定按钮**：绿色文字 "确定"，灰色按钮背景

**交互逻辑**：
- 点击"确定"或对话框外区域 → 关闭对话框
- ESC键 → 关闭对话框

---

## 五、动画效果

### 背景动画
- **云朵飘动**：缓慢横向移动

### 角色动画
- **僵尸手掌动画**：
  - 从地下升起： assets/effect/reanim/Zombie_hand.reanim

### UI动画
- **按钮高亮**：颜色渐变过渡（0.2秒）
- **对话框弹出**：缩放+淡入效果
- **对话框关闭**：缩放+淡出效果

---

## 七、资源清单

所有需要的动画资源都在这个文件定义了：

assets/effect/reanim/SelectorScreen.reanim

## 墓碑按钮

冒险模式：SelectorScreen_Adventure_highlight
生存模式的按钮对应的图片是：SelectorScreen_Vasebreaker_button.  
玩玩小游戏: SelectorScreen_Survival_button. 
解谜模式:  SelectorScreen_Challenges_highlight 


reanim: 
没解锁时，要显示类似轨道，而且点击后弹出未解锁的对话框
<name>SelectorScreen_Challenges_shadow</name>

### 对话框图片资源：


assets/images/dialog_bigbottomleft.png
assets/images/dialog_bigbottommiddle.png
assets/images/dialog_bigbottomright.png
assets/images/dialog_bottomleft.png
assets/images/dialog_bottommiddle.png
assets/images/dialog_bottomright.png
assets/images/dialog_centerleft.png
assets/images/dialog_centermiddle.png
assets/images/dialog_centerright.png
assets/images/dialog_header.png
assets/images/dialog_topleft.png
assets/images/dialog_topmiddle.png
assets/images/dialog_topright.png

### 花瓶样的按钮

屏幕右下角有3个花瓶样的按钮，鼠标悬浮时：
*   **选项 (Options)**
*   **帮助 (Help)**
*   **退出 (Quit)**

assets/images/SelectorScreen_Help1.png
assets/images/SelectorScreen_Help2.png
assets/images/SelectorScreen_Options1.png
assets/images/SelectorScreen_Options2.png
assets/images/SelectorScreen_Quit1.png
assets/images/SelectorScreen_Quit2.png

### 蒙版图片

PNG 文件是 8-bit 调色板模式，是用作 Alpha 蒙版的。使用 PNG 作为蒙版来抠出 JPG 中相应要显示的部分, 具体是让背景图片在蒙版的非白色部分透明，蒙版中有灰度值用于抗锯齿，添加预乘 alpha 处理来改善边缘质量。
assets/reanim/SelectorScreen_BG_Left.jpg
assets/reanim/SelectorScreen_BG_Right.jpg
assets/reanim/SelectorScreen_BG_Center.jpg

assets/reanim/SelectorScreen_BG_Left_.png
assets/reanim/SelectorScreen_BG_Right_.png
assets/reanim/SelectorScreen_BG_Center_.png

### 
---

## 八、交互流程图

```
启动游戏
    ↓
加载主菜单场景
    ↓
播放背景音乐
    ↓
显示主界面
    ↓
    ├─→ 鼠标悬停按钮 → 按钮高亮 → 播放音效
    │
    ├─→ 点击"冒险模式" → 进入当前关卡
    │
    ├─→ 点击"玩玩小游戏"
    │   ├─ 已解锁 → 进入小游戏
    │   └─ 未解锁 → 显示提示对话框
    │
    ├─→ 点击"解谜模式"
    │   ├─ 已解锁 → 进入解谜
    │   └─ 未解锁 → 显示提示对话框
    │
    ├─→ 点击"生存模式"
    │   ├─ 已解锁 → 进入生存模式
    │   └─ 未解锁 → 显示提示对话框
    │
    ├─→ 点击"选项" → 打开设置对话框
    │
    ├─→ 点击"帮助" → 打开帮助对话框
    │
    └─→ 点击"退出" → 退出游戏
```

### 更新日志

| 版本 | 日期 | 修改内容 | 作者 |
|------|------|---------|------|
| v1.0 | 2025-11-01 | 初始版本，基于截图分析完成 | Claude Code |

---

**文档状态**：✅ 已完成
**下一步行动**：创建对应 Epic/Story 规划实现工作
