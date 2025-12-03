# **2. Tech Stack (技术栈)**

为了确保项目的稳定性和可复现性，所有开发都将基于以下明确定义的技术栈。

## **Cloud Infrastructure (云基础设施)**
*   **Provider:** N/A (本地PC应用)
*   **Key Services:** N/A
*   **Deployment Regions:** N/A

## **Technology Stack Table (技术栈详情表)**
| Category | Technology | Version | Purpose | Rationale |
| :--- | :--- | :--- | :--- | :--- |
| **Language** | Go | latest stable | 主要开发语言 | 性能高、编译快、跨平台能力强。 |
| **Game Engine** | Ebitengine | latest stable | 2D游戏渲染与交互 | 纯Go实现，API简洁，跨平台支持良好。 |
| **Data Format** | YAML | v3 | 游戏数据配置 | 相比JSON更易于人类阅读和编写。 |
| **Go YAML Lib**| gopkg.in/yaml.v3 | latest stable | YAML文件解析 | Go社区广泛使用的YAML解析库。 |
| **Build Tool** | Go Modules | N/A | 依赖管理 | Go官方标准的依赖管理工具。 |
| **Testing** | Go Testing Pkg | N/A | 单元/集成测试 | Go标准库内置的测试框架。 |
| **UI Library**| Ebitengine Core API| N/A | UI渲染与交互 | 不引入`ebitenui`。手动管理UI布局能更精确地实现复刻目标。|
