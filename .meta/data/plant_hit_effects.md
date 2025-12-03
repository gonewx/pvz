# 植物受击与攻击效果汇总

根据 `reanim` 和 `particles` 资源文件的分析，以下是植物在被攻击（受击/被啃食）以及攻击僵尸时的动画与粒子效果汇总。

## 1. 植物受击/被啃食效果 (Hit/Eaten Effects)

这类效果通常发生在植物被僵尸啃食或受到伤害时。

| 植物名称 | 效果类型 | 资源文件 | 描述 |
| :--- | :--- | :--- | :--- |
| **坚果墙 (Wallnut)** | **粒子效果** | `data/particles/WallnutEatLarge.xml`<br>`data/particles/WallnutEatSmall.xml` | `data/particles/WallnutEatSmall.xml` 是在啃食时触发的， `data/particles/WallnutEatLarge.xml` 是在受损时更换图片时触发的。而且现在显示的位置也不对，应该是显示在啃食的接触点 |
| | **外观破损** | `assets/reanim/Wallnut_cracked1.png`<br>`assets/reanim/Wallnut_cracked2.png` | 根据生命值降低，外观会呈现轻微破损和严重破损状态。 |

在啃食时， 坚果墙就切换到 anim_blink_twitch动画，不摇摆了。 而且啃食时颜色要发亮，造成一闪一闪的效果。隔一小会儿就播放一次眨眼动画 anim_blink_twice 或anim_blink_thrice  随机选择
| **高坚果 (Tallnut)** | **粒子效果** | `data/particles/TallNutBlock.xml` | 推测在成功阻挡（如撑杆僵尸跳跃）或被攻击时触发的粒子效果。 |
| **南瓜头 (Pumpkin)** | **外观破损** | `assets/reanim/Pumpkin_damage2.png`<br>`assets/reanim/Pumpkin_damage3.png` | 类似于坚果墙，根据生命值降低呈现不同程度的破损外观。 |

> **注**：大多数其他植物在被啃食时没有特殊的视觉反馈（除了通用的啃食音效），直到生命值归零后消失。

## 2. 植物攻击/命中效果 (Attack/Impact Effects)

这类效果发生在植物发射的投射物命中僵尸时。

| 植物名称 | 效果类型 | 资源文件 | 描述 |
| :--- | :--- | :--- | :--- |
| **豌豆射手 (Peashooter)** | **命中粒子** | `data/particles/PeaSplat.xml` | 豌豆命中僵尸时产生的绿色汁液飞溅效果。 |
| **寒冰射手 (SnowPea)** | **命中粒子** | `data/particles/SnowPeaSplat.xml` | 冰豌豆命中僵尸时产生的冰渣飞溅效果。 |
| | **发射粒子** | `data/particles/SnowPeaPuff.xml` | 发射冰豌豆时枪口的冷气效果。 |
| **卷心菜投手 (CabbagePult)** | **命中粒子** | `data/particles/CabbageSplat.xml` | 卷心菜命中僵尸时产生的叶片碎裂效果。 |
| **玉米投手 (KernelPult)** | **命中粒子** | `data/particles/ButterSplat.xml` | 黄油命中僵尸时产生的黄油飞溅效果（玉米粒可能无特殊效果或共用）。 |
| **西瓜投手 (MelonPult)** | **命中粒子** | `data/particles/MelonImpact.xml` | 西瓜命中僵尸时产生的红色果肉飞溅效果。 |
| **冰瓜投手 (WinterMelon)** | **命中粒子** | `data/particles/WinterMelonImpact.xml` | 冰西瓜命中僵尸时产生的蓝色冰冻果肉飞溅效果。 |
| **星星果 (Starfruit)** | **命中粒子** | `data/particles/StarSplat.xml` | 星星命中僵尸时产生的黄色光点飞溅效果。 |
| **小喷菇 (PuffShroom)** | **命中粒子** | `data/particles/PuffSplat.xml` | 孢子命中僵尸时产生的紫色烟雾效果。 |
| | **发射粒子** | `data/particles/PuffShroomMuzzle.xml` | 发射孢子时的枪口烟雾效果。 |
| **大喷菇 (FumeShroom)** | **攻击粒子** | `data/particles/FumeCloud.xml`<br>`data/particles/GloomCloud.xml` | 喷出的气泡烟雾效果（GloomCloud 可能用于曾哥）。 |

## 3. 植物死亡/爆炸效果 (Death/Explosion Effects)

这类效果发生在植物死亡或触发一次性攻击时。

| 植物名称 | 效果类型 | 资源文件 | 描述 |
| :--- | :--- | :--- | :--- |
| **土豆雷 (PotatoMine)** | **爆炸粒子** | `data/particles/PotatoMine.xml` | 土豆雷爆炸时的泥土和烟雾效果。 |
| | **特殊外观** | `assets/reanim/PotatoMine_mashed.png` | 可能是被压扁或爆炸后的残留外观。 |
| **樱桃炸弹 (CherryBomb)** | **爆炸粒子** | `data/particles/BossExplosion.xml` | 代码中引用的爆炸效果（`BossExplosion`），产生巨大的黑色烟雾。 |
| **火爆辣椒 (Jalapeno)** | **爆炸粒子** | `data/particles/FireballDeath.xml` (推测) | 或者是通用的火焰效果，用于产生整行的火焰伤害。 |
| **毁灭菇 (DoomShroom)** | **爆炸粒子** | `data/particles/Doom.xml` | 毁灭菇爆炸时产生的巨大蘑菇云效果。 |

## 4. 其他特殊效果

- **种植效果**: `data/particles/Planting.xml` (种植时的泥土飞溅)。
- **花盆/睡莲**: `data/particles/PottedPlantGlow.xml` (发光效果)。
- **模仿者 (Imitater)**: `data/particles/ImitaterMorph.xml` (变形时的烟雾)。

## 总结

植物的视觉反馈主要集中在以下几个方面：
1.  **防御类植物**（坚果、南瓜）拥有**外观破损**状态和被啃食时的**碎屑粒子**。
2.  **射手/投手类植物**拥有丰富的**命中粒子**（Splat/Impact），增强打击感。
3.  **爆炸类植物**拥有专属的**爆炸粒子**效果。
4.  大多数植物共享通用的种植和死亡（消失）逻辑，没有单独的死亡粒子文件。
