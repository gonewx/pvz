package config

// ShadowSize 阴影尺寸配置
type ShadowSize struct {
	Width  float64 // 阴影宽度（像素）
	Height float64 // 阴影高度（像素）
}

// ShadowSizes 各类型实体的阴影尺寸
// 基于原版 PVZ 的视觉效果调整
// Story 10.7 修复：增大阴影尺寸以提高可见性
var ShadowSizes = map[string]ShadowSize{
	// 植物 - 增大尺寸以匹配原版效果
	"peashooter":    {Width: 50, Height: 25},
	"sunflower":     {Width: 55, Height: 28},
	"wallnut":       {Width: 70, Height: 35},
	"cherrybomb":    {Width: 60, Height: 30},
	"repeater":      {Width: 50, Height: 25},
	"snowpea":       {Width: 50, Height: 25},
	"chomper":       {Width: 60, Height: 30},
	"potatomine":    {Width: 45, Height: 22},
	"squash":        {Width: 65, Height: 32},
	"jalapeno":      {Width: 55, Height: 28},
	"spikeweed":     {Width: 60, Height: 30},
	"spikerock":     {Width: 65, Height: 32},
	"tallnut":       {Width: 75, Height: 38},
	"puffshroom":    {Width: 45, Height: 22},
	"sunshroom":     {Width: 50, Height: 25},
	"fumeshroom":    {Width: 55, Height: 28},
	"gravebuster":   {Width: 55, Height: 28},
	"hypnoshroom":   {Width: 55, Height: 28},
	"scaredyshroom": {Width: 50, Height: 25},
	"iceshroom":     {Width: 55, Height: 28},
	"doomshroom":    {Width: 60, Height: 30},
	"lilypad":       {Width: 65, Height: 32},
	"threepeater":   {Width: 60, Height: 30},
	"tanglekelp":    {Width: 50, Height: 25},
	"seashroom":     {Width: 50, Height: 25},
	"plantern":      {Width: 55, Height: 28},
	"cactus":        {Width: 50, Height: 25},
	"blover":        {Width: 55, Height: 28},
	"splitpea":      {Width: 50, Height: 25},
	"starfruit":     {Width: 55, Height: 28},
	"pumpkin":       {Width: 70, Height: 35},
	"magnetshroom":  {Width: 55, Height: 28},
	"cabbagepult":   {Width: 60, Height: 30},
	"kernelpult":    {Width: 60, Height: 30},
	"melonpult":     {Width: 65, Height: 32},
	"gatlingpea":    {Width: 55, Height: 28},
	"torchwood":     {Width: 60, Height: 30},
	"cobcannon":     {Width: 90, Height: 45},
	"garlic":        {Width: 55, Height: 28},
	"umbrellaleaf":  {Width: 60, Height: 30},
	"marigold":      {Width: 55, Height: 28},

	// 僵尸 - 增大尺寸
	"zombie":            {Width: 60, Height: 30},
	"zombie_cone":       {Width: 60, Height: 30},
	"zombie_bucket":     {Width: 60, Height: 30},
	"zombie_flag":       {Width: 60, Height: 30},
	"zombie_pole":       {Width: 65, Height: 32},
	"zombie_door":       {Width: 70, Height: 35},
	"zombie_newspaper":  {Width: 60, Height: 30},
	"zombie_football":   {Width: 75, Height: 38},
	"zombie_dancer":     {Width: 60, Height: 30},
	"zombie_backup":     {Width: 60, Height: 30},
	"zombie_snorkel":    {Width: 60, Height: 30},
	"zombie_zomboni":    {Width: 100, Height: 50},
	"zombie_sled":       {Width: 60, Height: 30},
	"zombie_dolphin":    {Width: 60, Height: 30},
	"zombie_jack":       {Width: 60, Height: 30},
	"zombie_balloon":    {Width: 60, Height: 30},
	"zombie_digger":     {Width: 60, Height: 30},
	"zombie_pogo":       {Width: 60, Height: 30},
	"zombie_yeti":       {Width: 70, Height: 35},
	"zombie_bungee":     {Width: 60, Height: 30},
	"zombie_ladder":     {Width: 65, Height: 32},
	"zombie_catapult":   {Width: 90, Height: 45},
	"zombie_gargantuar": {Width: 100, Height: 50},
	"zombie_imp":        {Width: 50, Height: 25},
	"boss_zombot":       {Width: 150, Height: 75},
}

// DefaultShadowSize 默认阴影尺寸（未配置时使用）
// Story 10.7 修复：增大默认尺寸
var DefaultShadowSize = ShadowSize{Width: 55, Height: 28}

// GetShadowSize 获取指定类型实体的阴影尺寸
// 参数:
//   - entityType: 实体类型字符串（如 "peashooter", "zombie"）
//
// 返回:
//   - 对应的阴影尺寸，如果未配置则返回默认尺寸
func GetShadowSize(entityType string) ShadowSize {
	if size, ok := ShadowSizes[entityType]; ok {
		return size
	}
	return DefaultShadowSize
}

// DefaultShadowAlpha 默认阴影透明度
// 值为 0.65，即 65% 透明度，符合原版 PVZ 的视觉效果
const DefaultShadowAlpha float32 = 0.65

// ========== 阴影位置偏移配置（可手工调节） ==========

// PlantShadowOffsetY 植物阴影 Y 偏移量（像素）
// 相对于格子底部中心的偏移
// 负值向上移动，正值向下移动
// 调整此值可以微调植物阴影的垂直位置
// 建议值范围：-20.0 ~ 0.0
const PlantShadowOffsetY float64 = -25.0

// ZombieShadowOffsetX 僵尸阴影 X 偏移量（像素）
// 相对于僵尸中心位置的偏移
// 负值向左移动，正值向右移动
// 调整此值可以微调僵尸阴影的水平位置
// 建议值范围：-30.0 ~ 30.0
const ZombieShadowOffsetX float64 = 10.0

// ZombieShadowOffsetY 僵尸阴影 Y 偏移量（像素）
// 相对于格子底部中心的偏移
// 负值向上移动，正值向下移动
// 调整此值可以微调僵尸阴影的垂直位置
// 建议值范围：-25.0 ~ 0.0
const ZombieShadowOffsetY float64 = -25.0

// ZombieBaseCenterOffsetY 僵尸基准 CenterOffsetY（像素）
// 用于阴影位置计算：不同僵尸类型（如路障/铁桶）由于装备向上延伸，
// 导致 CenterOffsetY 不同，需要基于基准值校正阴影位置。
// 此值取自普通僵尸的 CenterOffsetY。
const ZombieBaseCenterOffsetY float64 = 66.0
