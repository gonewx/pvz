# **6. Source Tree (源代码树)**

项目将采用以下目录结构，以确保清晰的关注点分离（Separation of Concerns），并为未来的扩展提供便利。

```plaintext
pvz/
├── main.go                 # 游戏主入口文件

├── assets/                   # 所有游戏资源
│   ├── images/               # 图片资源 (spritesheets)
│   ├── audio/                # 音频资源 (music, sfx)
│   └── fonts/                # 字体文件

├── data/                     # 外部化的游戏数据
│   ├── levels/               # 关卡配置文件 (e.g., level_1-1.yaml)
│   └── units/                # 单位属性文件 (e.g., plants.yaml, zombies.yaml)

├── pkg/                      # 项目核心代码库
│   ├── components/           # ECS: 所有组件的定义 (e.g., position.go, health.go)
│   │
│   ├── entities/             # ECS: 实体的定义和工厂函数 (e.g., plant_factory.go)
│   │
│   ├── systems/              # ECS: 所有系统的实现 (e.g., render_system.go)
│   │
│   ├── scenes/               # 游戏场景 (e.g., main_menu_scene.go, game_scene.go)
│   │
│   ├── ecs/                  # ECS框架的核心实现 (EntityManager)
│   │
│   ├── game/                 # 游戏的核心管理器 (e.g., scene_manager.go, game_state.go)
│   │
│   ├── utils/                # 通用工具函数 (e.g., timer.go)
│   │
│   └── config/               # 游戏配置加载与管理

├── go.mod                    # Go module文件
└── go.sum

```
