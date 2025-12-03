# **Epic 1: 游戏基础框架与主循环 (Game Foundation & Main Loop)**
**史诗目标:** 搭建整个Go+Ebitengine项目的基本结构，创建一个可以运行的空窗口，并实现游戏的核心状态管理和主菜单。这是所有后续功能的基础。

---
**Story 1.1: 项目初始化与窗口创建**
> **As a** 开发者,
> **I want** to set up the Go project structure and initialize an Ebitengine application,
> **so that** I have a running window which will serve as the canvas for the game.

**Acceptance Criteria:**
1.  项目遵循Go Modules的標準結構。
2.  Ebitengine被成功添加為項目依賴。
3.  運行 `go run .` 指令後，螢幕上會顯示一個固定大小（例如800x600像素）的空白視窗。
4.  視窗標題應設置為“植物大戰殭屍 - Go復刻版”。
5.  可以通過點擊視窗的關閉按鈕正常退出應用程式。

---
**Story 1.2: 游戏状态机与场景管理**
> **As a** 开发者,
> **I want** to implement a basic game state machine,
> **so that** the game can switch between different scenes like 'Main Menu' and 'In-Game'.

**Acceptance Criteria:**
1.  存在一個遊戲狀態管理器（例如 `SceneManager`）。
2.  定義了至少兩種場景狀態：`MainMenuScene` 和 `GameScene`。
3.  遊戲啟動時，默認進入並顯示 `MainMenuScene`。
4.  `SceneManager` 提供了切換場景的功能（例如 `SwitchToScene(...)`）。
5.  每個場景都有自己的 `Update` 和 `Draw` 邏輯。

---
**Story 1.3: 资源管理器框架**
> **As a** 开发者,
> **I want** to create a resource manager that can load image and audio files,
> **so that** game assets can be centrally managed and accessed by any part of the game.

**Acceptance Criteria:**
1.  存在一個資源管理器（例如 `ResourceManager`），在遊戲啟動時初始化。
2.  可以成功加載一個PNG格式的圖片文件（例如主菜單背景）並在場景中使用。
3.  可以成功加載一個音頻文件（例如主菜單背景音樂）並循環播放。
4.  如果資源加載失敗，遊戲應能打印錯誤日誌而不是直接崩潰。
5.  資源應只被加載一次，並在內存中重複使用。

---
**Story 1.4: 主菜单UI与交互**
> **As a** 玩家,
> **I want** to see and interact with the main menu,
> **so that** I can start the game or exit.

**Acceptance Criteria:**
1.  主菜單場景 (`MainMenuScene`) 必須顯示正確的背景圖片。
2.  主菜單必須顯示“開始冒險”和“退出遊戲”兩個按鈕（暫時可以是文字或簡單圖形）。
3.  鼠標懸停在按鈕上時，按鈕有視覺變化（例如變色或放大）。
4.  點擊“開始冒險”按鈕後，遊戲狀態會通過 `SceneManager` 切換到 `GameScene`。
5.  點擊“退出遊戲”按鈕後，應用程式會正常關閉。
