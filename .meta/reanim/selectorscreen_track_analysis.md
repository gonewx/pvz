# SelectorScreen.reanim 轨道分析报告

**生成时间**: 自动生成
**总轨道数**: 48
**总帧数**: 706 帧 @ 20 FPS = 35.3 秒

---

## 1. 轨道类型统计

- **动画定义轨道**: 14 个（控制时间窗口）
- **混合轨道**: 33 个（有图片+位置+f值）
- **图片轨道**: 1 个（只有图片）
- **逻辑轨道**: 0 个（只有位置）

## 2. 动画定义轨道（时间线）

| 轨道名 | 可见窗口 | 持续时间 | 用途推测 |
|--------|---------|---------|---------|
| anim_open | 0-12 | 0.65秒 | 开场阶段定义（墓碑通过按钮Y坐标变化升起） |
| anim_sign | 13-40 | 1.4秒 | 木牌显示控制 |
| anim_idle | 41-77 | 1.9秒 | 主菜单循环？ |
| anim_grass | 78-102 | 1.2秒 | 草丛晃动控制 |
| anim_flower2 | 103-160 | 2.9秒 | 花朵2晃动控制 |
| anim_flower1 | 161-178 | 0.9秒 | 花朵1晃动控制 |
| anim_flower3 | 179-197 | 0.9秒 | 花朵3晃动控制 |
| anim_start | 全隐藏 | 0秒 | 开始过渡？（未使用） |
| anim_cloud1 | 198-335 | 6.9秒 | 云朵1飘动控制 |
| anim_cloud7 | 336-421 | 4.3秒 | 云朵7飘动控制 |
| anim_cloud2 | 422-502 | 4.0秒 | 云朵2飘动控制 |
| anim_cloud4 | 503-705 | 10.2秒 | 云朵4飘动控制 |
| anim_cloud6 | 569-638 | 3.5秒 | 云朵6飘动控制 |
| anim_cloud5 | 639-705 | 3.4秒 | 云朵5飘动控制 |

## 3. 混合轨道（可见元素）

| 轨道名 | 可见窗口 | 持续时间 | 元素类型 |
|--------|---------|---------|---------|
| Cloud1 | 198-335 | 6.9秒 | 云朵1 |
| Cloud7 | 336-421 | 4.3秒 | 云朵7 |
| Cloud2 | 422-502 | 4.0秒 | 云朵2 |
| Cloud4 | 503-568 | 3.3秒 | 云朵4 |
| Cloud6 | 569-638 | 3.5秒 | 云朵6 |
| Cloud5 | 639-705 | 3.4秒 | 云朵5 |
| SelectorScreen_BG_Center | 全程 | 35.3秒 | 装饰 |
| SelectorScreen_BG_Left | 全程 | 35.3秒 | 装饰 |
| SelectorScreen_BG_Right | 全程 | 35.3秒 | 装饰 |
| almanac_key_shadow | 全程 | 35.3秒 | 按钮 |
| SelectorScreen_Adventure_shadow | 全程 | 35.3秒 | 按钮 |
| SelectorScreen_Adventure_button | 全程 | 35.3秒 | 按钮 |
| SelectorScreen_Survival_shadow | 全程 | 35.3秒 | 按钮 |
| SelectorScreen_Survival_button | 全程 | 35.3秒 | 按钮 |
| SelectorScreen_Challenges_shadow | 全程 | 35.3秒 | 按钮 |
| SelectorScreen_Challenges_button | 全程 | 35.3秒 | 按钮 |
| SelectorScreen_ZenGarden_shadow | 全程 | 35.3秒 | 按钮 |
| SelectorScreen_ZenGarden_button | 全程 | 35.3秒 | 按钮 |
| SelectorScreen_StartAdventure_shadow | 全程 | 35.3秒 | 按钮 |
| SelectorScreen_StartAdventure_button | 全程 | 35.3秒 | 按钮 |
| leaf3 | 78-102 | 1.2秒 | 树叶 |
| leaf2 | 78-102 | 1.2秒 | 树叶 |
| leaf22 | 78-102 | 1.2秒 | 树叶 |
| leaf_SelectorScreen_Leaves | 78-102 | 1.2秒 | 树叶 |
| leaf4 | 78-102 | 1.2秒 | 树叶 |
| leaf5 | 78-102 | 1.2秒 | 树叶 |
| leaf1 | 78-102 | 1.2秒 | 树叶 |
| flower1 | 179-197 | 0.9秒 | 花朵1 |
| flower2 | 103-160 | 2.9秒 | 花朵2 |
| flower3 | 161-178 | 0.9秒 | 花朵3 |
| woodsign1 | 13-77 | 3.2秒 | 装饰 |
| woodsign2 | 13-77 | 3.2秒 | 装饰 |
| woodsign3 | 13-77 | 3.2秒 | 装饰 |

## 4. 关键发现

### 4.1 anim_idle 的问题 ⚠️

**现状**：
- anim_idle 可见窗口：帧 41-77（共 38 帧，约 1.9 秒）
- 这个窗口太短，无法覆盖所有装饰元素：
  - 云朵最早在帧 198 出现（9.9秒后）
  - 草丛在帧 78-103 出现（3.9-5.2秒）
  - 花朵在帧 103-198 出现（5.2-9.9秒）

**推测**：
- anim_idle 可能不是"主循环动画"
- 可能是墓碑升起后的"短暂等待状态"（1.9秒）
- 原版可能有其他机制来播放完整的 706 帧

**当前实现**：
- 播放所有 706 帧作为主循环
- 从第 20 帧开始（跳过墓碑升起的帧 0-16）
- 循环到第 706 帧后回到第 20 帧

### 4.2 anim_open 的实现机制

**发现**：
- anim_open 轨道在帧 0-12 可见（f=0 默认值），帧 13 开始隐藏（f=-1）
- 墓碑升起动画通过按钮轨道的 Y 坐标变化实现：
  - 帧 0: y=638.6（地下）
  - 帧 1-12: Y 坐标逐渐减小
  - 帧 12: y=79.7（最终位置）
- anim_open 只是时间窗口定义，不直接控制位置

### 4.3 时间轴总览

```
帧数   0 ---- 13 -- 20 -- 41 -- 78 -- 103 - 161 - 179 - 198 -------- 336 -- 422 --- 503 -- 569 --- 639 --- 706
       |      |     |     |     |     |     |     |     |            |      |       |      |        |       |
事件   墓碑   木牌  主    anim  草丛  花2   花3   花1   云1飘动----+  云7-+ 云2--+ 云4--+ 云6---+ 云5--+
       升起   开始  循环  idle  晃动
                    开始  窗口

- 墓碑升起：一次性（不循环）
- 主循环：帧 20-706，周期 34.3 秒
- 云朵/草丛/花朵：在各自窗口内周期性出现
```

## 5. 实现建议

### 5.1 当前实现 ✅

```go
// 墓碑升起（anim_open）：只播放一次
PlayAnimationNoLoop(entity, "anim_open")

// 主循环：播放帧 20-706
if anim_open.IsFinished {
    PlayAnimation(entity, "anim_idle")  // 构建 MergedTracks
    reanimComp.VisibleFrameCount = 706   // 覆盖为全帧
    reanimComp.CurrentFrame = 20         // 从第 20 帧开始

    // 循环检查：防止回到第 0 帧
    if reanimComp.CurrentFrame < 20 {
        reanimComp.CurrentFrame = 20
    }
}
```

### 5.2 VisibleTracks 白名单

**需要加入白名单的轨道**（一直显示）：
- `SelectorScreen_BG*` - 背景
- `SelectorScreen_*_button` - 按钮（根据解锁状态）
- `SelectorScreen_*_shadow` - 按钮阴影（根据解锁状态）
- `Cloud1-7` 和 `anim_cloud1-7` - 云朵
- `leaf*` - 树叶
- `woodsign*` - 木牌

**不加入白名单的轨道**（根据 f 值周期性显示）：
- `anim_grass` - 草丛（在帧 78-102 自动显示）
- `flower1-3` - 花朵（触发显示，暂未实现）

## 6. 参考数据

### 完整轨道列表

**动画定义轨道**（14个）：
| 轨道名 | 可见窗口 | 持续时间 |
|--------|---------|---------|
| anim_open | 0-12 | 0.65秒 |
| anim_sign | 13-40 | 1.4秒 |
| anim_idle | 41-77 | 1.9秒 |
| anim_grass | 78-102 | 1.2秒 |
| anim_flower2 | 103-160 | 2.9秒 |
| anim_flower1 | 161-178 | 0.9秒 |
| anim_flower3 | 179-197 | 0.9秒 |
| anim_start | 全隐藏 | 0秒 |
| anim_cloud1 | 198-335 | 6.9秒 |
| anim_cloud7 | 336-421 | 4.3秒 |
| anim_cloud2 | 422-502 | 4.0秒 |
| anim_cloud4 | 503-705 | 10.2秒 |
| anim_cloud6 | 569-638 | 3.5秒 |
| anim_cloud5 | 639-705 | 3.4秒 |
