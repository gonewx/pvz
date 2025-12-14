package components

// PlantTagComponent 植物标签组件（ECS 标签模式）
//
// 所有植物实体都应添加此组件，用于统一识别植物实体。
// 这是一个空结构体，仅作为标签使用，不存储任何数据。
//
// 使用场景：
//   - 战斗状态序列化时查询所有植物实体
//   - 任何需要区分植物和其他实体的逻辑
//
// 优势：
//   - 新增植物类型时无需修改查询逻辑，只需在工厂函数中添加此组件
//   - 符合 ECS 架构的组件标识模式
//   - 避免基于 PlantComponent 的硬编码查询
type PlantTagComponent struct{}
