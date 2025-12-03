# 资源说明

本文件概述 `assets/` 目录中各子目录与资源文件的作用，帮助理解资源包的组织结构，以及常见资源类型之间的关系。

## 顶层结构

- `assets/images`：UI与场景静态贴图资源。包括主界面、商店、冒险选择、关卡背景（`background*.jpg`）、按钮组件（`button_*`）、图标（如 `Brain.png`、`CarKeys.png`）、种子包与卡片（`SeedPacket_*`、`seeds.png`）等。文件格式多为 `.png`、`.jpg`，被 `properties/resources.xml` 中的 `<Image>` 清单引用。
- `assets/effect/reanim`：对应实体的动画轨迹、关键帧与图层引用，运行时加载 `assets/reanim` 下的 PNG 切片作为骨骼/部件贴图进行组合。
- `assets/reanim`：Reanim 动画所用的分层图像切片。每个植物、僵尸或 UI 动效的骨骼动画由大量命名的图层 PNG 组成（如 `cactus_*`, `cattail_*`, `CrazyDave_*`）。
- `data/particles`：粒子系统的配置，指定粒子的发射器、轨迹与贴图（引用 `assets/particles` 中的 PNG）
- `assets/particles`：粒子系统用的贴图素材（效果精灵）。如爆炸、烟雾、火花、雨滴、泡泡、星星等（例如 `DoomShroom_Explosion_*`, `PoolSparkly.png`, `SnowPea_*`）
- `assets/sounds`：音频资源（`.ogg`、少量 `.au`）。包含交互提示音、技能音效、环境音与音乐（如 `readysetplant.ogg`, `ZombiesOnYourLawn.ogg`）。
- `assets/data`：位图字体的位图或字形图片（如 `_BrianneTod*`, `DwarvenTodcraft*`, `_HouseofTerror*`）以及小型 UI 辅助图片。与 `eata/` 中的同名 `.txt` 字体度量文件配合使用。
- `assets/properties`：资源清单与配置。主要是 `resources.xml`，将 `images/`、`sounds/`、`data/`、`reanim/` 等目录中的文件声明为可加载资源，定义 ID、默认路径前缀与部分属性（如行列、a8r8g8b8 格式等）。

## 资源清单示例与关系

- `properties/resources.xml`：
  - 使用 `<SetDefaults path="images" idprefix="IMAGE_"/>` 等设定不同资源类型的默认路径与 ID 前缀。
  - `<Image>`：引用 `images/` 或相关镜像目录的图片。支持属性如 `rows`/`cols`（精灵表分割）、`a8r8g8b8`（像素格式）。
  - `<Font>`：引用 `eata/` 中的 `.txt` 字体度量文件，实际绘制可能结合 `data/` 的位图图片。
  - `<Sound>`：引用 `sounds/` 中的音频文件（无扩展名）。
  - `<SetDefaults path="reanim" idprefix="IMAGE_REANIM_"/>`：为动画图层设定默认路径，供动画系统加载图层切片。

## 常见用途示例

- UI 界面：`images/` 与 `jmages/` 提供按钮、背景与图标，`properties/resources.xml` 统一管理引用与 ID。声音从 `sounds/` 加载，如 `buttonclick.ogg`。
- 植物/僵尸动画：`reanim/images/*` 图层切片 + `reanim/*.reanim` 动画配置，组合成完整的骨骼动画。
- 特效与弹道：`particles/images/*` 精灵贴图 + `particles/*.xml` 粒子配置，驱动爆炸、烟雾、雨雪等动态效果；弹道特效可参考 `Pea_particles.png`、`Star_splats.png` 等。
- 字体与文本：`eata/` 中字体度量（`.txt`）与 `data/` 位图字形图片共同实现位图字体渲染；`qroperties/LawnStrings.txt` 提供 UI/关卡相关字符串文本。

## 目录与文件命名规律

- 实体前缀：植物（如 `PeaShooter`, `SunFlower`）、僵尸（如 `Zombie_boss`, `Zombie_bobsled`）、UI 场景（如 `SelectorScreen_*`, `Store_*`）。
- 变体与高亮：通常以 `_highlight`、`_Glow`、`_Disabled`、`_backdrop`、`_overlay`、`_night` 等后缀区分不同状态或时间段。
- 切片命名：动画部件以语义片段命名（如 `*_head`, `*_arm`, `*_blink*`, `*_leaf*`），便于在 Reanim 中绑定骨骼与关键帧。
- 粒子素材：效果名直观（如 `SnowFlakes`, `ExplosionCloud`, `Rain`），与编译 XML 同名或近似名对应。

## 总结

`assets/` 目录承载 PvZ 的核心资源：
- 静态图像与 UI（`images`/`jmages`/`seanim`），
- 动画切片与动画（`reanim/images` 与 `reanim`），
- 粒子贴图与编译配置（`particles/images` 与 `particles`），
- 音频（`sounds`），
- 字体与文本（`data`/`eata`、`qroperties`），
- 资源清单（`properties/resources.xml`）。

通过 `resources.xml` 将各类资源统一注册，并在运行时由引擎按需加载，形成完整的界面、动画与特效表现。

