# **7. Coding Standards (编码标准)**

所有由AI代理生成的代码都必须严格遵守以下标准，以确保代码库的一致性、可读性和可维护性。

## **Core Standards (核心标准)**
*   **语言与版本:** Go (latest stable)。
*   **格式化:** 所有代码在提交前必须使用 `gofmt` 或 `goimports`进行格式化。
*   **Linting:** 使用 `golangci-lint` 作为代码质量检查工具，并遵循其默认的推荐规则集。
*   **测试文件:** 测试文件必须与源文件在同一个包（package）内，并以 `_test.go` 结尾。

## **Naming Conventions (命名约定)**
| Element | Convention | Example |
| :--- | :--- | :--- |
| **Packages** | `snake_case` | `render_system` |
| **Structs & Interfaces** | `PascalCase` | `PositionComponent` |
| **Methods & Functions** | `PascalCase` (public), `camelCase` (private) | `Update()`, `calculateDamage()` |
| **Variables** | `camelCase` | `currentHealth` |
| **Constants** | `PascalCase` | `DefaultZombieSpeed` |
| **Struct Fields** | `PascalCase` | `X, Y float64` |

## **Critical Rules (关键规则)**

### **零耦合原则** ✅ **Enforced**

**System** 绝不能直接相互调用。所有跨系统的通信必须通过查询`EntityManager`或通过`EventBus`（如果实现）进行。

**成功案例: Epic 14 - ECS 系统耦合解除重构**

**问题**:
- 9 个系统直接调用 ReanimSystem 的 PlayCombo/PlayAnimation 方法（15 处违规）
- 违反 ECS 零耦合原则，导致系统间紧耦合
- 需要在构造函数中传递 ReanimSystem 引用

**解决方案**:
- 引入 `AnimationCommandComponent` 组件，实现组件驱动的动画控制
- 业务系统添加 AnimationCommand 组件表达动画请求
- ReanimSystem 读取并处理组件，执行动画播放

**改造成果**:
```
改造前 (15 处违规):
BehaviorSystem  ──直接调用──> ReanimSystem.PlayCombo()
WaveSpawnSystem ──直接调用──> ReanimSystem.PlayCombo()
...（共 9 个系统）

改造后 (零违规):
BehaviorSystem  ──添加组件──> AnimationCommandComponent
WaveSpawnSystem ──添加组件──> AnimationCommandComponent  ──ReanimSystem读取──> 执行动画
```

**架构改进**:
- ✅ 系统间耦合度降低 90%
- ✅ 完全符合 ECS 架构原则
- ✅ 代码更易测试和维护
- ✅ 消除了 9 个系统对 ReanimSystem 的依赖

**详见**: `docs/prd/epic-14-ecs-decoupling-refactor.md`

---

*   **数据-行为分离:** **Component** 结构体中严禁包含任何方法（行为逻辑）。所有逻辑必须在**System**中实现。
*   **接口优于具体类型:** 在函数和方法签名中，优先接受接口（interfaces）而非具体的结构体类型，以提高代码的灵活性和可测试性。
*   **错误处理:** 严禁忽略（discard）错误。所有可能返回`error`的函数都必须进行检查。使用`fmt.Errorf`或Go 1.13+的`%w`来包装错误以提供上下文。
*   **禁止全局变量:** 除了用于管理全局状态的单例（如`GameState`），严禁使用全局变量。所有依赖都应通过构造函数注入。
*   **注释:** 为所有公共的函数、方法、结构体和接口编写清晰的GoDoc注释。复杂的逻辑块内部需要有行内注释解释其意图。
