
- 草坪背景不是通过缩放来适应窗口,而是游戏中只显示中间部分,游戏开始前要有一个从左到右的滑动效果显
示整体场景的动画 . 是从左滑到右，再从右滑动回中间。
- 阳光计数器的背景和金色矩形框也已经在bar5 上绘制好了,只是在正确的位置显示正确的数值即可
- 铲子和铲子槽位画在了卡片槽位上，要移到右方
- 阳光计数值的显示位置 在哪里设置 ?

## story 3.2

- 要参考 @2.2.story.md 中阳光计数器的位置计算 方式，使用相对位置方式确定卡片的位置
- 卡片还有重叠，卡片槽每个卡槽中间是有边距的，查看截图
- 卡片大于卡槽了，需要增加常量用来控制卡片显示的缩放因子。而且卡片上已经有固定的阳光值了，不需要再显示。


### correct-course
- 半透明的植物图像不是跟随鼠标，是显示在将要被种植的位置，跟随鼠标的图像应该是不透明的。
- 我将原图片目录备份到了 assets/imagesold， 现在的资源目录 assets 收集了完整的原版资源，请先根据 assets/properties/resources.xml 结合我们调整后的实际目录，创建我们的资源映射文件（由原来的 xml 改为yaml格式） ，并创新新的资源引用与管理模块，然后将代码原有引用旧资源的逻辑改为使用新的资源管理方式。

## 5.3

- 还有2个问题，卡片会渲染在植物和僵尸上面。坚果墙一放，帽子僵尸帽子就会消失，是被攻击了吗？

## aniamtion migration

/mnt/disk0/project/game/pvz/ck/pvzwine_reverse 这个项目是对原版游戏资源数据的研究与解析， 请结合测试程序，将我们之前实现的基于动画帧的动画系统，改为使用和原版游戏同样的动画系统。 解析后的资源已经放在 assets/reanim， assets/effect/reanim。

## 粒子效果

现在要全面支持粒子效果， 例如：僵尸的手臂掉落、头掉落的效果等等，请全面分析粒子系统的配置格式，优雅的实现粒子系统。实现方案要优先使用Ebitengine 引擎，有必要的话先使用 `mcp__deepwiki` 工具的`ask_question`方法，查阅最新的文档，以找到最正确的方法

- `assets/effect/particles`：粒子系统的配置，指定粒子的发射器、轨迹与贴图（引用 `assets/particles` 中的 PNG）
- `assets/particles`：粒子系统用的贴图素材（效果精灵）。如爆炸、烟雾、火花、雨滴、泡泡、星星等（例如 `DoomShroom_Explosion_*`, `PoolSparkly.png`, `SnowPea_*`）

- 僵尸头粒子效果和原版的表现不一样,请参考我的理解,检查实现是否不正确? .meta/particles/ZombieHead.md  
assets/effect/particles/ZombieHead.xml 从实际运行 cmd/particles/main.go 看,僵尸头没有在地面弹跳的效果

-  僵尸头粒子效果和原版的表现不一样, 掉落速度慢,头滚动少,很快就消失了，没看到反弹地面的效果.是不是我们对配置文件的字段含义理解有误?或实现有误? .meta/particles/ZombieHead.md  assets/effect/particles/ZombieHead.xml  cmd/particles/main.go

## 8

- 现在项目的基础功能全部实现，需要根据 @.meta/whitepaper.md 的说明一步步实现关卡， 注意说明文档是大模型生成的，有可能不准确，有和你对游戏理解不一致的地方，明确提出，和我沟通确认。
