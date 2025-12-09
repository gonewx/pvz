# **6. Source Tree (源代码树)**

项目采用以下目录结构，确保清晰的关注点分离（Separation of Concerns），支持跨平台构建和未来扩展。

> **最后更新**: 2025-12-09

---

## 项目根目录

```plaintext
pvz/
├── main.go                          # 游戏主入口文件
├── embed.go                         # Go embed 资源嵌入配置
├── go.mod / go.sum                  # Go 模块依赖管理
├── Makefile                         # 跨平台构建配置
├── README.md                        # 项目说明文档
├── CLAUDE.md                        # Claude AI 开发上下文
├── AGENTS.md                        # AI Agent 配置
├── versioninfo.json                 # 版本信息（Windows 资源）
├── resource_windows_*.syso          # Windows 图标资源文件
│
├── pkg/                             # 核心代码库
├── cmd/                             # 调试和验证工具
├── internal/                        # 内部模块（解析器）
├── data/                            # 游戏配置和数据
├── assets/                          # 游戏资源文件
├── docs/                            # 项目文档
├── scripts/                         # 构建和部署脚本
├── mobile/                          # 移动平台适配
└── build/                           # 编译输出目录
```

---

## pkg/ - 核心代码库

```plaintext
pkg/
├── app/                             # 应用主类
│   └── app.go                       # 游戏窗口和场景管理器
│
├── ecs/                             # ECS 框架核心（泛型实现）
│   ├── entity_manager.go            # 实体/组件管理
│   ├── generics.go                  # 泛型 API 实现
│   └── entity_manager_test.go       # 单元测试
│
├── components/                      # ECS 组件定义（80+ 个，纯数据）
│   ├── 核心组件
│   │   ├── position.go              # 位置组件
│   │   ├── velocity.go              # 速度组件
│   │   ├── scale.go                 # 缩放组件
│   │   ├── health.go                # 生命值组件
│   │   ├── armor.go                 # 护甲组件
│   │   ├── collision.go             # 碰撞组件
│   │   ├── clickable.go             # 可点击组件
│   │   ├── lifetime.go              # 生命周期组件
│   │   └── behavior.go              # 行为类型组件
│   │
│   ├── 动画组件
│   │   ├── reanim_component.go      # Reanim 骨骼动画组件
│   │   ├── animation_command.go     # 动画命令队列
│   │   └── squash_animation_component.go  # 压扁动画组件
│   │
│   ├── 游戏逻辑组件
│   │   ├── plant.go                 # 植物组件
│   │   ├── plant_card.go            # 植物卡片组件
│   │   ├── sun.go                   # 阳光组件
│   │   ├── wave_timer.go            # 波次计时器
│   │   ├── zombie_target_lane.go    # 僵尸目标行
│   │   ├── zombie_wave_state.go     # 僵尸波次状态
│   │   ├── lawnmower_component.go   # 除草车组件
│   │   ├── conveyor_belt_component.go  # 传送带组件
│   │   ├── bowling_nut_component.go # 保龄球坚果组件
│   │   └── level_phase_component.go # 关卡阶段组件
│   │
│   ├── 粒子/特效组件
│   │   ├── particle_component.go    # 粒子组件
│   │   ├── emitter_component.go     # 发射器组件
│   │   ├── flash_effect_component.go  # 闪烁效果组件
│   │   └── shadow_component.go      # 阴影组件
│   │
│   ├── UI 组件
│   │   ├── ui_component.go          # 基础 UI 组件
│   │   ├── button_component.go      # 按钮组件
│   │   ├── dialog_component.go      # 对话框组件
│   │   ├── slider_component.go      # 滑块组件
│   │   ├── checkbox_component.go    # 复选框组件
│   │   ├── text_input_component.go  # 文本输入组件
│   │   ├── virtual_keyboard_component.go  # 虚拟键盘组件
│   │   ├── tooltip_component.go     # 提示框组件
│   │   ├── plant_selection_component.go  # 选卡界面组件
│   │   ├── plant_preview.go         # 植物预览组件
│   │   ├── pause_menu_component.go  # 暂停菜单组件
│   │   └── level_progress_bar_component.go  # 进度条组件
│   │
│   └── 特殊组件
│       ├── camera_component.go      # 摄像机组件
│       ├── difficulty_component.go  # 难度组件
│       ├── spawn_constraint.go      # 生成约束组件
│       ├── opening_animation_component.go  # 开场动画组件
│       ├── dave_dialogue_component.go  # 戴夫对话组件
│       ├── guided_tutorial_component.go  # 引导教程组件
│       ├── reward_animation_component.go  # 奖励动画组件
│       ├── reward_card_component.go # 奖励卡片组件
│       ├── reward_panel_component.go  # 奖励面板组件
│       ├── final_wave_warning_component.go  # 最终波提示组件
│       └── zombies_won_phase_component.go  # 游戏失败阶段组件
│
├── systems/                         # ECS 系统实现（80+ 个，纯逻辑）
│   ├── 核心系统
│   │   ├── input_system.go          # 输入处理系统
│   │   ├── physics_system.go        # 物理系统
│   │   ├── render_system.go         # 渲染系统
│   │   ├── lifetime_system.go       # 生命周期系统
│   │   ├── camera_system.go         # 摄像机系统
│   │   └── lawn_grid_system.go      # 草坪网格系统
│   │
│   ├── Reanim 动画系统
│   │   ├── reanim_system.go         # 核心骨骼动画系统
│   │   ├── reanim_update.go         # 动画更新逻辑
│   │   ├── reanim_helpers.go        # 动画辅助函数
│   │   └── render_reanim.go         # 动画渲染
│   │
│   ├── 粒子系统
│   │   └── particle_system.go       # 粒子生成/更新/渲染
│   │
│   ├── 游戏逻辑系统
│   │   ├── level_system.go          # 关卡管理系统
│   │   ├── level_phase_system.go    # 关卡阶段系统
│   │   ├── sun_spawn_system.go      # 阳光生成系统
│   │   ├── sun_movement_system.go   # 阳光移动系统
│   │   ├── sun_collection_system.go # 阳光收集系统
│   │   ├── wave_spawn_system.go     # 波次生成系统
│   │   ├── wave_timing_system.go    # 波次计时系统
│   │   ├── lawnmower_system.go      # 除草车系统
│   │   ├── bowling_nut_system.go    # 保龄球系统
│   │   ├── conveyor_belt_system.go  # 传送带系统
│   │   ├── shovel_interaction_system.go  # 铲子交互系统
│   │   └── sodding_system.go        # 铺草皮系统
│   │
│   ├── 僵尸/难度系统
│   │   ├── difficulty_engine.go     # 难度���擎
│   │   ├── spawn_constraint_system.go  # 生成约束系统
│   │   ├── lane_allocator.go        # 行分配算法
│   │   ├── zombie_groan_system.go   # 僵尸呻吟系统
│   │   └── zombie_lane_transition_system.go  # 僵尸行切换系统
│   │
│   ├── 警告/提示系统
│   │   ├── flag_wave_warning_system.go  # 旗帜波警告系统
│   │   ├── final_wave_warning_system.go # 最终波警告系统
│   │   └── flash_effect_system.go   # 闪烁效果系统
│   │
│   ├── 教程/动画系统
│   │   ├── tutorial_system.go       # 教程系统
│   │   ├── guided_tutorial_system.go  # 引导教程系统
│   │   ├── opening_animation_system.go  # 开场���画系统
│   │   ├── readysetplant_system.go  # Ready-Set-Plant 系统
│   │   ├── dave_dialogue_system.go  # 戴夫对话系统
│   │   └── zombies_won_phase_system.go  # 游戏失败流程系统
│   │
│   ├── 奖励系统
│   │   ├── reward_animation_system.go  # 奖励动画系统
│   │   └── reward_panel_render_system.go  # 奖励面板渲染
│   │
│   ├── UI 系统
│   │   ├── plant_card_system.go     # 植物卡片逻辑
│   │   ├── plant_card_render_system.go  # 植物卡片渲染
│   │   ├── plant_preview_system.go  # 植物预览逻辑
│   │   ├── plant_preview_render_system.go  # 植物预览渲染
│   │   ├── plant_selection_system.go  # 选卡界面系统
│   │   ├── button_system.go         # 按钮逻辑系统
│   │   ├── button_render_system.go  # 按钮渲染系统
│   │   ├── slider_system.go         # 滑块系统
│   │   ├── checkbox_system.go       # 复选框系统
│   │   ├── text_input_system.go     # 文本输入系统
│   │   ├── text_input_render_system.go  # 文本输入渲染
│   │   ├── virtual_keyboard_system.go  # 虚拟键盘系统
│   │   ├── virtual_keyboard_render_system.go  # 虚拟键盘渲染
│   │   ├── dialog_input_system.go   # 对话框输入系统
│   │   ├── dialog_render_system.go  # 对话框渲染系统
│   │   ├── pause_menu_render_system.go  # 暂停菜单渲染
│   │   └── level_progress_bar_render_system.go  # 进度条渲染
│   │
│   └── behavior/                    # 行为子系统
│       └── (行为树实现)
│
├── entities/                        # 实体工厂函数（30+ 个）
│   ├── plant_factory.go             # 植物实体工厂
│   ├── zombie_factory.go            # 僵尸实体工厂
│   ├── projectile_factory.go        # 投射物实体工厂
│   ├── effect_factory.go            # 特效实体工厂
│   ├── particle_factory.go          # 粒子实体工厂
│   ├── sun_factory.go               # 阳光实体工厂
│   ├── lawnmower_factory.go         # 除草车实体工厂
│   ├── bowling_nut_factory.go       # 保龄球坚果工厂
│   ├── dave_factory.go              # 戴夫实体工厂
│   ├── zombie_hand_factory.go       # 僵尸手实体工厂
│   ├── selector_screen_factory.go   # 选卡界面工厂
│   ├── dialog_factory.go            # 对话框工厂
│   ├── game_over_dialog_factory.go  # 游戏结束对话框工厂
│   ├── button_factory.go            # 按钮工厂
│   ├── virtual_keyboard_factory.go  # 虚拟键盘工厂
│   ├── ui_factory.go                # UI 实体工厂
│   ├── plant_card_factory.go        # 植物卡片工厂
│   ├── user_management_dialog_factory.go  # 用户管理对话框工厂
│   ├── new_user_dialog_factory.go   # 新用户对话框工厂
│   ├── rename_user_dialog_factory.go  # 重命名用户对话框工厂
│   ├── delete_user_dialog_factory.go  # 删除用户对话框工厂
│   ├── reanim_helpers.go            # Reanim 辅助函数
│   └── test_helpers.go              # 测试辅助函数
│
├── scenes/                          # 游戏场景（20+ 个文件）
│   ├── scene.go                     # 基础场景接口
│   ├── game_scene.go                # 游戏场景核心
│   ├── game_scene_init.go           # 游戏场景初始化
│   ├── game_scene_background.go     # 游戏场景背景
│   ├── game_scene_ui.go             # 游戏场景 UI
│   ├── game_scene_effects.go        # 游戏场景特效
│   ├── game_scene_conveyor.go       # 传送带关卡场景
│   ├── main_menu_scene.go           # 主菜单场景
│   ├── main_menu_buttons.go         # 主菜单按钮
│   └── main_menu_zombie_hand.go     # 主菜单僵尸手动画
│
├── game/                            # 游戏核心管理器
│   ├── game_state.go                # 全局游戏状态
│   ├── scene_manager.go             # 场景管理器
│   ├── resource_manager.go          # 资源管理器
│   ├── resource_config.go           # 资源配置
│   ├── audio_manager.go             # 音频管理器
│   ├── save_manager.go              # 存档管理器
│   ├── settings_manager.go          # 设置管理器
│   ├── plant_unlock_manager.go      # 植物解锁管理器
│   ├── battle_save_data.go          # 战斗存档数据结构
│   ├── battle_serializer.go         # 战斗状态序列化
│   ├── scene.go                     # 场景基类
│   └── lawn_strings.go              # 草坪字符串常量
│
├── config/                          # 配置加载与管理（30+ 个文件）
│   ├── level_config.go              # 关卡配置
│   ├── spawn_rules.go               # 僵尸生成规则
│   ├── zombie_stats.go              # 僵尸属性数据
│   ├── zombie_physics.go            # 僵尸物理参数
│   ├── plant_config.go              # 植物配置
│   ├── plant_card_config.go         # 植物卡片配置
│   ├── reanim_config.go             # Reanim 动画配置
│   ├── reanim_config_manager.go     # Reanim 配置管理器
│   ├── shadow_config.go             # 阴影配置
│   ├── particle_anchor_config.go    # 粒子锚点配置
│   ├── ui_config.go                 # UI 配置
│   ├── menu_config.go               # 菜单配置
│   ├── layout_config.go             # 布局配置
│   ├── loading_config.go            # 加载配置
│   ├── reward_panel_config.go       # 奖励面板配置
│   ├── level_progress_bar_config.go # 进度条配置
│   ├── gameover_door_config.go      # 游戏结束门配置
│   ├── sodding_config.go            # 铺草皮配置
│   └── unit_config.go               # 单位配置
│
├── modules/                         # 功能模块（UI 模块化）
│   ├── pause_menu_module.go         # 暂停菜单模块
│   ├── help_panel_module.go         # 帮助面板模块
│   ├── options_panel_module.go      # 选项面板模块
│   ├── settings_panel_module.go     # 设置面板模块
│   └── plant_selection_module.go    # 选卡界面模块
│
├── types/                           # 类型定义
│   ├── plant_type.go                # 植物类型枚举
│   └── zombie_type.go               # 僵尸类型枚举
│
├── utils/                           # 工具库（20+ 个文件）
│   ├── coordinates.go               # 坐标转换库
│   ├── grid_utils.go                # 网格工具
│   ├── reanim_loader.go             # Reanim 加载器
│   ├── root_motion.go               # 根运动计算
│   ├── bitmap_font.go               # 位图字体
│   ├── nine_patch.go                # 九宫格图片
│   ├── image_utils.go               # 图像工具
│   ├── text_utils.go                # 文本工具
│   ├── input.go                     # 输入处理
│   ├── easing.go                    # 缓动函数
│   ├── platform.go                  # 平台检测
│   ├── platform_mobile.go           # 移动平台
│   ├── storage_android.go           # Android 存储
│   └── storage_default.go           # 默认存储
│
└── embedded/                        # 嵌入式资源管理
    └── (资源加载适配)
```

---

## internal/ - 内部模块

```plaintext
internal/
├── reanim/                          # Reanim 动画解析器
│   ├── parser.go                    # XML 解析器
│   ├── types.go                     # 数据类型定义
│   └── parser_test.go               # 单元测试
│
├── particle/                        # 粒子系统解析器
│   ├── parser.go                    # XML 解析器
│   ├── types.go                     # 数据类型定义
│   ├── value_parser.go              # 值解析器
│   └── parser_test.go               # 单元测试
│
└── audio/                           # 音频解析器
    └── au_decoder.go                # AU 格式解码器
```

---

## cmd/ - 调试和验证工具

```plaintext
cmd/
├── analyze_reanim/                  # Reanim 动画分析工具
├── analyze_skew/                    # 倾斜分析工具
├── animation_showcase/              # 动画展示工具
├── calculate_hitbox/                # 碰撞箱计算工具
├── particles/                       # 粒子系统测试工具
├── test_textinput/                  # 文本输入测试
├── verify_bowling/                  # 保龄球验证
├── verify_gameplay/                 # 游戏流程验证
├── verify_level_rewards/            # 关卡��励验证
├── verify_opening/                  # 开场动画验证
├── verify_pause_menu/               # 暂停菜单验证
├── verify_readysetplant/            # Ready-Set-Plant 验证
├── verify_reward_animation/         # 奖励动画验证
├── verify_reward_card/              # 奖励卡验证
├── verify_reward_panel/             # 奖励面板验证
└── verify_zombies_won/              # 游戏失败验证
```

---

## data/ - 游戏数据和配置

```plaintext
data/
├── levels/                          # 关卡配置文件
│   ├── level-1-1.yaml               # 关卡 1-1（教学关）
│   ├── level-1-2.yaml               # 关卡 1-2
│   ├── level-1-3.yaml               # 关卡 1-3
│   ├── level-1-4.yaml               # 关卡 1-4
│   └── level-1-5.yaml               # 关卡 1-5（保龄球）
│
├── reanim/                          # Reanim 骨骼动画定义（120+ 个）
│   ├── 植物动画
│   │   ├── PeaShooter.reanim        # 豌豆射手
│   │   ├── Sunflower.reanim         # 向日葵
│   │   ├── WallNut.reanim           # 坚果墙
│   │   ├── CherryBomb.reanim        # 樱桃炸弹
│   │   └── ... (70+ 种植物)
│   │
│   ├── 僵尸动画
│   │   ├── Zombie.reanim            # 基础僵尸
│   │   ├── Zombie_conehead.reanim   # 路障僵尸
│   │   ├── Zombie_buckethead.reanim # 铁桶僵尸
│   │   ├── Zombie_charred.reanim    # 烧焦僵尸
│   │   ├── LawnMoweredZombie.reanim # 被除草车碾压僵尸
│   │   └── ... (40+ 种僵尸)
│   │
│   └── 特效/UI 动画
│       ├── CrazyDave.reanim         # 疯狂戴夫
│       ├── FinalWave.reanim         # 最终波
│       ├── StartReadySetPlant.reanim  # Ready-Set-Plant
│       ├── SelectorScreen.reanim    # 选卡界面
│       ├── Sun.reanim               # 阳光
│       └── ... (其他)
│
├── reanim_config/                   # Reanim 动画配置（130+ 个）
│   ├── peashooter.yaml              # 豌豆射手配置
│   ├── sunflower.yaml               # 向日葵配置
│   ├── zombie.yaml                  # 僵尸配置
│   └── ... (更多配置)
│
├── particles/                       # 粒子效果配置（100+ 个 XML）
│   ├── Planting.xml                 # 种植效果
│   ├── PeaSplat.xml                 # 豌豆溅射
│   ├── MelonImpact.xml              # 西瓜撞击
│   ├── FireballDeath.xml            # 火焰死亡
│   ├── ZombieRise.xml               # 僵尸出现
│   ├── ZombieHead.xml               # 僵尸头部
│   ├── MowerCloud.xml               # 除草车烟雾
│   ├── Award.xml                    # 奖励效果
│   └── ... (��多粒子)
│
├── reanim_config.yaml               # 全局 Reanim 配置
├── spawn_rules.yaml                 # 僵尸生成规则
├── zombie_stats.yaml                # 僵尸属性数据
└── zombie_physics.yaml              # 僵尸物理参数
```

---

## assets/ - 游戏资源

```plaintext
assets/
├── images/                          # 图像资源（PNG）
│   ├── 植物精灵图
│   ├── 僵尸精灵图
│   ├── UI 元素
│   └── 背景和装饰
│
├── sounds/                          # 音频资源
│   ├── bgm/                         # 背景音乐
│   └── sfx/                         # 音效
│
├── reanim/                          # Reanim 资源副本
│
├── particles/                       # 粒子配置副本
│
├── fonts/                           # 字体文件
│
├── icons/                           # 应用图标（多平台）
│   ├── windows/                     # Windows .ico + .png
│   ├── macos/                       # macOS icon.iconset
│   ├── linux/                       # Linux 多尺寸 PNG
│   ├── ios/                         # iOS AppIcon.appiconset
│   ├── android/                     # Android mipmap
│   └── web/                         # Web favicon + PWA
│
└── properties/                      # 属性配置
```

---

## scripts/ - 构建脚本

```plaintext
scripts/
├── build-apk.sh                     # Android APK 构建脚本
├── sign-apk.sh                      # APK 签名脚本
├── rsync.sh                         # 文件同步脚本
├── test_animation_showcase.sh       # 动画展示测试脚本
├── Info.plist                       # macOS 应用配置
├── pvz.desktop                      # Linux 桌面入口文件
└── wasm_index.html                  # WebAssembly 页面模板
```

---

## mobile/ - 移动平台适配

```plaintext
mobile/
├── mobile.go                        # 移动平台接口
├── stub.go                          # 存根实现
├── embed.go                         # 资源嵌入
├── assets/                          # 移动平台资源
└── data/                            # 移动平台数据
```

---

## 项目统计

| 类别 | 数量 |
|------|------|
| Go 源文件 | 300+ |
| 组件类型 | 80+ |
| 系统类型 | 80+ |
| 实体工厂 | 30+ |
| 配置文件 | 30+ |
| 动画文件 (.reanim) | 120+ |
| 粒子配置 (.xml) | 100+ |
| 关卡配置 | 5 |
| 文档文件 | 150+ |
| 总代码行数 | 200,000+ |
