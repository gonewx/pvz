package components

import (
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// DialogComponent 对话框组件
// 用于显示模态对话框（如未解锁提示、确认对话框等）
type DialogComponent struct {
	Title            string         // 对话框标题（如"未解锁！"）
	Message          string         // 对话框消息（如"进行更多新冒险..."）
	Buttons          []DialogButton // 按钮列表（如["确定"]）
	Parts            *DialogParts   // 九宫格图片资源
	IsVisible        bool           // 是否可见
	Width            float64        // 对话框宽度
	Height           float64        // 对话框高度
	ChildEntities    []ecs.EntityID // 关联的子实体（如输入框），对话框销毁时一起销毁
	AutoClose        bool           // 点击按钮后是否自动关闭对话框（默认 true，新用户对话框设为 false）
	Modal            bool           // 是否为模态对话框（true=点击遮罩不关闭，阻止背景交互）
	HoveredButtonIdx int            // 当前悬停的按钮索引（-1 表示没有悬停）- ECS 架构重构
	PressedButtonIdx int            // 当前按下的按钮索引（-1 表示没有按下）- 用于按下效果
	UseBigBottom     bool           // 是否使用大底部区域（用于两行按钮布局）

	// Story 10.9: 音效状态跟踪
	LastPressedButtonIdx int // 上一帧按下的按钮索引（用于检测按下状态变化，播放音效）
}

// DialogButton 对话框按钮
type DialogButton struct {
	Label       string        // 按钮文字
	OnClick     func()        // 点击回调
	X           float64       // 按钮相对对话框的 X 坐标
	Y           float64       // 按钮相对对话框的 Y 坐标
	Width       float64       // 按钮宽度
	Height      float64       // 按钮高度
	LeftImage   *ebiten.Image // 按钮左边图片
	MiddleImage *ebiten.Image // 按钮中间图片（可拉伸）
	RightImage  *ebiten.Image // 按钮右边图片
	MiddleWidth float64       // 中间部分宽度

	// Story 10.9: 音效支持
	ClickSoundID   string // 点击释放时播放的音效ID（如 "SOUND_BUTTONCLICK"）
	PressedSoundID string // 按下时播放的音效ID（如 "SOUND_GRAVEBUTTON"）
}

// DialogParts 九宫格对话框资源
// 包含所有用于渲染对话框的图片资源
type DialogParts struct {
	// 四个边角（固定大小，不拉伸）
	TopLeft     *ebiten.Image // dialog_topleft.png
	TopRight    *ebiten.Image // dialog_topright.png
	BottomLeft  *ebiten.Image // dialog_bottomleft.png
	BottomRight *ebiten.Image // dialog_bottomright.png

	// 四个边缘（单向拉伸）
	TopMiddle    *ebiten.Image // dialog_topmiddle.png
	BottomMiddle *ebiten.Image // dialog_bottommiddle.png
	CenterLeft   *ebiten.Image // dialog_centerleft.png
	CenterRight  *ebiten.Image // dialog_centerright.png

	// 中心区域（双向拉伸）
	CenterMiddle *ebiten.Image // dialog_centermiddle.png

	// 特殊装饰
	Header *ebiten.Image // dialog_header.png (骷髅头)

	// 大对话框的额外部分（可选）
	BigBottomLeft   *ebiten.Image // dialog_bigbottomleft.png
	BigBottomMiddle *ebiten.Image // dialog_bigbottommiddle.png
	BigBottomRight  *ebiten.Image // dialog_bigbottomright.png
}
