package game

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// SaveData 保存数据结构
//
// Story 8.6: 关卡进度保存系统
//
// 保存内容：
//   - 最高完成关卡（如 "1-3" 表示完成了 1-3，可以玩 1-4）
//   - 解锁的植物列表
//   - 解锁的工具列表
type SaveData struct {
	HighestLevel   string   `yaml:"highestLevel"`   // 最高完成关卡ID，如 "1-3"
	UnlockedPlants []string `yaml:"unlockedPlants"` // 已解锁植物ID列表
	UnlockedTools  []string `yaml:"unlockedTools"`  // 已解锁工具ID列表，如 ["shovel"]
	HasStartedGame bool     `yaml:"hasStartedGame"` // 是否已开始过游戏（用于区分新用户和老用户）
}

// UserMetadata 用户元数据
//
// Story 12.4: 用户管理 UI
//
// 存储用户的基本信息和时间戳
type UserMetadata struct {
	Username    string    `yaml:"username"`    // 用户名
	CreatedAt   time.Time `yaml:"createdAt"`   // 创建时间
	LastLoginAt time.Time `yaml:"lastLoginAt"` // 最后登录时间
}

// UserListData 用户列表数据
//
// Story 12.4: 用户管理 UI
//
// 存储所有用户的元数据和当前登录用户
type UserListData struct {
	Users       []UserMetadata `yaml:"users"`       // 所有用户列表
	CurrentUser string         `yaml:"currentUser"` // 当前登录的用户名
}

// SaveManager 保存管理器
//
// 职责：
//   - 加载和保存游戏进度
//   - 管理关卡解锁状态
//   - 管理植物和工具解锁状态
//   - 管理多用户存档（Story 12.4）
//
// 架构说明：
//   - 单例模式，全局唯一实例
//   - 数据持久化到本地文件（YAML格式，与项目其他配置文件保持一致）
//   - 由 GameState 调用，不直接与系统交互
type SaveManager struct {
	saveDir      string        // 存档目录
	userListPath string        // 用户列表文件路径
	currentUser  string        // 当前用户名
	data         *SaveData     // 当前用户的存档数据
	userList     *UserListData // 用户列表数据
}

// NewSaveManager 创建保存管理器
//
// 参数：
//   - saveDir: 保存文件目录路径（如 "data/saves"）
//
// 返回：
//   - *SaveManager: 新创建的保存管理器实例
//   - error: 如果创建失败返回错误
func NewSaveManager(saveDir string) (*SaveManager, error) {
	// 确保保存目录存在
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create save directory: %w", err)
	}

	sm := &SaveManager{
		saveDir:      saveDir,
		userListPath: filepath.Join(saveDir, "users.yaml"),
		currentUser:  "",
		data: &SaveData{
			HighestLevel:   "",
			UnlockedPlants: []string{},
			UnlockedTools:  []string{},
		},
		userList: &UserListData{
			Users:       []UserMetadata{},
			CurrentUser: "",
		},
	}

	// 尝试加载用户列表
	if err := sm.loadUserListFile(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load user list: %w", err)
		}
		// 文件不存在，使用默认空列表
	}

	// 如果有当前用户，加载其存档数据
	if sm.userList.CurrentUser != "" {
		sm.currentUser = sm.userList.CurrentUser
		if err := sm.Load(); err != nil {
			// 存档文件损坏或不存在，使用默认数据
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to load save data for user %s: %w", sm.currentUser, err)
			}
		}
	}

	return sm, nil
}

// loadUserListFile 从文件加载用户列表
func (sm *SaveManager) loadUserListFile() error {
	data, err := os.ReadFile(sm.userListPath)
	if err != nil {
		return err
	}

	var userList UserListData
	if err := yaml.Unmarshal(data, &userList); err != nil {
		return fmt.Errorf("failed to parse user list: %w", err)
	}

	sm.userList = &userList
	return nil
}

// saveUserListFile 保存用户列表到文件
func (sm *SaveManager) saveUserListFile() error {
	data, err := yaml.Marshal(sm.userList)
	if err != nil {
		return fmt.Errorf("failed to marshal user list: %w", err)
	}

	if err := os.WriteFile(sm.userListPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write user list file: %w", err)
	}

	return nil
}

// getSaveFilePath 获取用户存档文件路径
func (sm *SaveManager) getSaveFilePath(username string) string {
	return filepath.Join(sm.saveDir, username+".yaml")
}

// Load 从文件加载保存数据
//
// 返回：
//   - error: 如果加载失败返回错误（文件不存在返回 os.ErrNotExist）
func (sm *SaveManager) Load() error {
	if sm.currentUser == "" {
		return fmt.Errorf("no user selected")
	}

	saveFilePath := sm.getSaveFilePath(sm.currentUser)
	// 读取文件
	data, err := os.ReadFile(saveFilePath)
	if err != nil {
		return err
	}

	// 解析 YAML
	var saveData SaveData
	if err := yaml.Unmarshal(data, &saveData); err != nil {
		return fmt.Errorf("failed to parse save data: %w", err)
	}

	sm.data = &saveData
	return nil
}

// Save 保存数据到文件
//
// 返回：
//   - error: 如果保存失败返回错误
func (sm *SaveManager) Save() error {
	if sm.currentUser == "" {
		return fmt.Errorf("no user selected")
	}

	// 序列化为 YAML（格式化输出，便于人工阅读和调试）
	data, err := yaml.Marshal(sm.data)
	if err != nil {
		return fmt.Errorf("failed to marshal save data: %w", err)
	}

	// 写入文件
	saveFilePath := sm.getSaveFilePath(sm.currentUser)
	if err := os.WriteFile(saveFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write save file: %w", err)
	}

	return nil
}

// GetHighestLevel 获取最高完成关卡
//
// 返回：
//   - string: 最高完成关卡ID，如 "1-3"，空字符串表示未完成任何关卡
func (sm *SaveManager) GetHighestLevel() string {
	return sm.data.HighestLevel
}

// SetHighestLevel 设置最高完成关卡
//
// 只有当新关卡比当前记录更高时才更新
//
// 参数：
//   - levelID: 关卡ID，如 "1-3"
func (sm *SaveManager) SetHighestLevel(levelID string) {
	// 简单比较：只要levelID不为空，就更新
	// TODO: 实现关卡ID的大小比较（如 "1-4" > "1-3"）
	if levelID != "" {
		sm.data.HighestLevel = levelID
	}
}

// GetHasStartedGame 获取是否已开始过游戏的标记
//
// 返回：
//   - bool: true 表示用户已开始过游戏（显示 Adventure 按钮和关卡数字）
//     false 表示新用户（显示 StartAdventure 按钮，不显示关卡数字）
func (sm *SaveManager) GetHasStartedGame() bool {
	return sm.data.HasStartedGame
}

// SetHasStartedGame 设置已开始游戏的标记
//
// 当用户首次点击"开始冒险吧"按钮时调用
func (sm *SaveManager) SetHasStartedGame() {
	sm.data.HasStartedGame = true
}

// GetUnlockedPlants 获取已解锁植物列表
//
// 返回：
//   - []string: 已解锁植物ID列表（副本，修改不影响原数据）
func (sm *SaveManager) GetUnlockedPlants() []string {
	// 返回副本
	plants := make([]string, len(sm.data.UnlockedPlants))
	copy(plants, sm.data.UnlockedPlants)
	return plants
}

// UnlockPlant 解锁植物
//
// 参数：
//   - plantID: 植物ID，如 "sunflower"
func (sm *SaveManager) UnlockPlant(plantID string) {
	// 检查是否已解锁
	for _, p := range sm.data.UnlockedPlants {
		if p == plantID {
			return // 已解锁，无需重复添加
		}
	}

	// 添加到列表
	sm.data.UnlockedPlants = append(sm.data.UnlockedPlants, plantID)
}

// GetUnlockedTools 获取已解锁工具列表
//
// 返回：
//   - []string: 已解锁工具ID列表（副本，修改不影响原数据）
func (sm *SaveManager) GetUnlockedTools() []string {
	// 返回副本
	tools := make([]string, len(sm.data.UnlockedTools))
	copy(tools, sm.data.UnlockedTools)
	return tools
}

// UnlockTool 解锁工具
//
// 参数：
//   - toolID: 工具ID，如 "shovel"
func (sm *SaveManager) UnlockTool(toolID string) {
	// 检查是否已解锁
	for _, t := range sm.data.UnlockedTools {
		if t == toolID {
			return // 已解锁，无需重复添加
		}
	}

	// 添加到列表
	sm.data.UnlockedTools = append(sm.data.UnlockedTools, toolID)
}

// IsToolUnlocked 检查工具是否已解锁
//
// 参数：
//   - toolID: 工具ID，如 "shovel"
//
// 返回：
//   - bool: true 表示已解锁，false 表示未解锁
func (sm *SaveManager) IsToolUnlocked(toolID string) bool {
	for _, t := range sm.data.UnlockedTools {
		if t == toolID {
			return true
		}
	}
	return false
}

// --- 多用户管理方法 (Story 12.4) ---

// LoadUserList 加载所有用户列表
//
// 返回：
//   - []UserMetadata: 用户列表（按创建日期排序）
//   - error: 如果加载失败返回错误
func (sm *SaveManager) LoadUserList() ([]UserMetadata, error) {
	// 按创建日期排序
	users := make([]UserMetadata, len(sm.userList.Users))
	copy(users, sm.userList.Users)
	sort.Slice(users, func(i, j int) bool {
		return users[i].CreatedAt.Before(users[j].CreatedAt)
	})
	return users, nil
}

// GetCurrentUser 获取当前登录用户名
//
// 返回：
//   - string: 当前用户名，空字符串表示未登录
func (sm *SaveManager) GetCurrentUser() string {
	return sm.currentUser
}

// ValidateUsername 验证用户名合法性
//
// 规则：
//   - 不能为空
//   - 只能包含字母、数字、空格
//   - 长度限制 1-20 字符
//
// 参数：
//   - username: 用户名
//
// 返回：
//   - error: 如果验证失败返回错误
func (sm *SaveManager) ValidateUsername(username string) error {
	// 检查空用户名
	if username == "" {
		return fmt.Errorf("请输入你的名字，以创建新的用户档案。档案用于保存游戏积分和进度。")
	}

	// 检查长度
	if len(username) > 20 {
		return fmt.Errorf("用户名长度不能超过 20 个字符")
	}

	// 检查字符（只允许字母、数字、空格）
	matched, err := regexp.MatchString(`^[a-zA-Z0-9 ]+$`, username)
	if err != nil {
		return fmt.Errorf("failed to validate username: %w", err)
	}
	if !matched {
		return fmt.Errorf("用户名只能包含字母、数字和空格")
	}

	return nil
}

// CreateUser 创建新用户
//
// 参数：
//   - username: 用户名
//
// 返回：
//   - error: 如果创建失败返回错误
func (sm *SaveManager) CreateUser(username string) error {
	// 验证用户名
	if err := sm.ValidateUsername(username); err != nil {
		return err
	}

	// 检查用户是否已存在
	for _, user := range sm.userList.Users {
		if user.Username == username {
			return fmt.Errorf("用户名 '%s' 已存在", username)
		}
	}

	// 创建新用户元数据
	now := time.Now()
	newUser := UserMetadata{
		Username:    username,
		CreatedAt:   now,
		LastLoginAt: now,
	}

	// 添加到用户列表
	sm.userList.Users = append(sm.userList.Users, newUser)
	sm.userList.CurrentUser = username
	sm.currentUser = username

	// 创建默认存档数据
	sm.data = &SaveData{
		HighestLevel:   "",
		UnlockedPlants: []string{},
		UnlockedTools:  []string{},
		HasStartedGame: false,
	}

	// 保存用户列表和存档
	if err := sm.saveUserListFile(); err != nil {
		return fmt.Errorf("failed to save user list: %w", err)
	}

	if err := sm.Save(); err != nil {
		return fmt.Errorf("failed to save user data: %w", err)
	}

	return nil
}

// RenameUser 重命名用户
//
// 参数：
//   - oldName: 旧用户名
//   - newName: 新用户名
//
// 返回：
//   - error: 如果重命名失败返回错误
func (sm *SaveManager) RenameUser(oldName, newName string) error {
	// 验证新用户名
	if err := sm.ValidateUsername(newName); err != nil {
		return err
	}

	// 检查旧用户是否存在
	userIndex := -1
	for i, user := range sm.userList.Users {
		if user.Username == oldName {
			userIndex = i
			break
		}
	}
	if userIndex == -1 {
		return fmt.Errorf("用户 '%s' 不存在", oldName)
	}

	// 检查新用户名是否已存在
	for _, user := range sm.userList.Users {
		if user.Username == newName {
			return fmt.Errorf("用户名 '%s' 已存在", newName)
		}
	}

	// 重命名存档文件
	oldPath := sm.getSaveFilePath(oldName)
	newPath := sm.getSaveFilePath(newName)
	if err := os.Rename(oldPath, newPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to rename save file: %w", err)
		}
		// 文件不存在，仅更新元数据
	}

	// 更新用户列表
	sm.userList.Users[userIndex].Username = newName
	if sm.userList.CurrentUser == oldName {
		sm.userList.CurrentUser = newName
		sm.currentUser = newName
	}

	// 保存用户列表
	if err := sm.saveUserListFile(); err != nil {
		return fmt.Errorf("failed to save user list: %w", err)
	}

	return nil
}

// DeleteUser 删除用户
//
// 参数：
//   - username: 用户名
//
// 返回：
//   - error: 如果删除失败返回错误
func (sm *SaveManager) DeleteUser(username string) error {
	// 检查用户是否存在
	userIndex := -1
	for i, user := range sm.userList.Users {
		if user.Username == username {
			userIndex = i
			break
		}
	}
	if userIndex == -1 {
		return fmt.Errorf("用户 '%s' 不存在", username)
	}

	// 删除存档文件
	saveFilePath := sm.getSaveFilePath(username)
	if err := os.Remove(saveFilePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete save file: %w", err)
		}
		// 文件不存在，继续
	}

	// 从用户列表中移除
	sm.userList.Users = append(sm.userList.Users[:userIndex], sm.userList.Users[userIndex+1:]...)

	// 如果删除的是当前用户，清空当前用户
	if sm.currentUser == username {
		sm.currentUser = ""
		sm.userList.CurrentUser = ""
		sm.data = &SaveData{
			HighestLevel:   "",
			UnlockedPlants: []string{},
			UnlockedTools:  []string{},
			HasStartedGame: false,
		}
	}

	// 保存用户列表
	if err := sm.saveUserListFile(); err != nil {
		return fmt.Errorf("failed to save user list: %w", err)
	}

	return nil
}

// SwitchUser 切换用户
//
// 参数：
//   - username: 用户名
//
// 返回：
//   - error: 如果切换失败返回错误
func (sm *SaveManager) SwitchUser(username string) error {
	// 检查用户是否存在
	userExists := false
	var userIndex int
	for i, user := range sm.userList.Users {
		if user.Username == username {
			userExists = true
			userIndex = i
			break
		}
	}
	if !userExists {
		return fmt.Errorf("用户 '%s' 不存在", username)
	}

	// 保存当前用户存档（如果有）
	if sm.currentUser != "" {
		if err := sm.Save(); err != nil {
			return fmt.Errorf("failed to save current user data: %w", err)
		}
	}

	// 切换到新用户
	sm.currentUser = username
	sm.userList.CurrentUser = username

	// 更新最后登录时间
	sm.userList.Users[userIndex].LastLoginAt = time.Now()

	// 加载新用户存档
	if err := sm.Load(); err != nil {
		if os.IsNotExist(err) {
			// 存档不存在，使用默认数据
			sm.data = &SaveData{
				HighestLevel:   "",
				UnlockedPlants: []string{},
				UnlockedTools:  []string{},
				HasStartedGame: false,
			}
		} else {
			return fmt.Errorf("failed to load user data: %w", err)
		}
	}

	// 保存用户列表（更新最后登录时间）
	if err := sm.saveUserListFile(); err != nil {
		return fmt.Errorf("failed to save user list: %w", err)
	}

	return nil
}

// --- 战斗存档管理方法 (Story 18.1) ---

// BattleSaveFileSuffix 战斗存档文件后缀
const BattleSaveFileSuffix = "_battle.sav"

// GetBattleSavePath 获取用户的战斗存档文件路径
//
// 参数：
//   - username: 用户名
//
// 返回：
//   - string: 战斗存档文件完整路径，格式: {saveDir}/{username}_battle.sav
func (sm *SaveManager) GetBattleSavePath(username string) string {
	return filepath.Join(sm.saveDir, username+BattleSaveFileSuffix)
}

// HasBattleSave 检查用户是否有战斗存档
//
// 参数：
//   - username: 用户名
//
// 返回：
//   - bool: true 表示存在战斗存档，false 表示不存在
func (sm *SaveManager) HasBattleSave(username string) bool {
	battleSavePath := sm.GetBattleSavePath(username)
	_, err := os.Stat(battleSavePath)
	return err == nil
}

// GetBattleSaveInfo 获取战斗存档信息（预览）
//
// 读取存档文件的头部信息，无需加载完整的存档数据。
// 用于在主菜单显示存档预览信息。
//
// 参数：
//   - username: 用户名
//
// 返回：
//   - *BattleSaveInfo: 存档预览信息
//   - error: 如果读取失败返回错误
func (sm *SaveManager) GetBattleSaveInfo(username string) (*BattleSaveInfo, error) {
	battleSavePath := sm.GetBattleSavePath(username)

	// 使用 BattleSerializer 加载完整数据
	// 未来可优化为只读取头部信息
	serializer := NewBattleSerializer()
	saveData, err := serializer.LoadBattle(battleSavePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load battle save: %w", err)
	}

	return saveData.ToBattleSaveInfo(), nil
}

// DeleteBattleSave 删除用户的战斗存档
//
// 参数：
//   - username: 用户名
//
// 返回：
//   - error: 如果删除失败返回错误，文件不存在不视为错误
func (sm *SaveManager) DeleteBattleSave(username string) error {
	battleSavePath := sm.GetBattleSavePath(username)

	err := os.Remove(battleSavePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，不视为错误
			return nil
		}
		return fmt.Errorf("failed to delete battle save: %w", err)
	}

	return nil
}
