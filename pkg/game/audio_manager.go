package game

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// AudioType 音频类型
type AudioType int

const (
	// AudioTypeMusic 背景音乐（循环播放）
	AudioTypeMusic AudioType = iota
	// AudioTypeSound 音效（单次播放）
	AudioTypeSound
)

// AudioManager 音频管理器
// 职责：
//   - 统一管理游戏中所有音效和背景音乐的播放
//   - 实现音量控制（从 SettingsManager 读取设置）
//   - 提供便捷的播放接口
//
// 设计原则：
//   - 中心化管理：所有音频播放都通过 AudioManager
//   - 与设置联动：自动应用 SettingsManager 中的音量设置
//   - 简化调用：通过资源ID播放，无需关心路径
type AudioManager struct {
	resourceManager *ResourceManager  // 资源管理器（用于加载音频）
	settingsManager *SettingsManager  // 设置管理器（用于读取音量设置）
	soundPlayers    map[string]*audio.Player // 音效播放器缓存（资源ID -> 播放器）
	musicPlayers    map[string]*audio.Player // 背景音乐播放器缓存（资源ID -> 播放器）
	currentMusic    *audio.Player     // 当前播放的背景音乐
	currentMusicID  string            // 当前播放的背景音乐ID
}

// NewAudioManager 创建新的音频管理器
//
// 参数：
//   - rm: ResourceManager 实例（用于加载音频文件）
//   - sm: SettingsManager 实例（用于读取音量设置，可为 nil）
//
// 返回：
//   - *AudioManager: 音频管理器实例
func NewAudioManager(rm *ResourceManager, sm *SettingsManager) *AudioManager {
	return &AudioManager{
		resourceManager: rm,
		settingsManager: sm,
		soundPlayers:    make(map[string]*audio.Player),
		musicPlayers:    make(map[string]*audio.Player),
	}
}

// PlaySound 播放音效
// 音效使用 SoundVolume 设置控制音量，单次播放后停止
//
// 参数：
//   - soundID: 音效资源ID（如 "SOUND_BUTTONCLICK", "SOUND_PLANT"）
//
// 返回：
//   - bool: 是否成功播放
func (am *AudioManager) PlaySound(soundID string) bool {
	// 检查音效是否启用
	if am.settingsManager != nil {
		settings := am.settingsManager.GetSettings()
		if !settings.SoundEnabled {
			return false // 音效已禁用
		}
	}

	// 获取或加载音效播放器
	player := am.getSoundPlayer(soundID)
	if player == nil {
		return false
	}

	// 设置音量
	volume := am.getSoundVolume()
	player.SetVolume(volume)

	// 重置并播放
	if err := player.Rewind(); err != nil {
		log.Printf("[AudioManager] Warning: Failed to rewind sound %s: %v", soundID, err)
	}
	player.Play()

	return true
}

// PlayMusic 播放背景音乐
// 背景音乐使用 MusicVolume 设置控制音量，循环播放
// 同一时间只能播放一首背景音乐
//
// 参数：
//   - musicID: 音乐资源ID（如 "SOUND_WINMUSIC", "SOUND_LOSEMUSIC"）
//
// 返回：
//   - bool: 是否成功播放
func (am *AudioManager) PlayMusic(musicID string) bool {
	// 检查音乐是否启用
	if am.settingsManager != nil {
		settings := am.settingsManager.GetSettings()
		if !settings.MusicEnabled {
			return false // 音乐已禁用
		}
	}

	// 如果已经在播放同一首音乐，不重复播放
	if am.currentMusicID == musicID && am.currentMusic != nil && am.currentMusic.IsPlaying() {
		return true
	}

	// 停止当前音乐
	am.StopMusic()

	// 获取或加载音乐播放器
	player := am.getMusicPlayer(musicID)
	if player == nil {
		return false
	}

	// 设置音量
	volume := am.getMusicVolume()
	player.SetVolume(volume)

	// 重置并播放
	if err := player.Rewind(); err != nil {
		log.Printf("[AudioManager] Warning: Failed to rewind music %s: %v", musicID, err)
	}
	player.Play()

	am.currentMusic = player
	am.currentMusicID = musicID

	log.Printf("[AudioManager] Playing music: %s (volume: %.2f)", musicID, volume)

	return true
}

// StopMusic 停止当前背景音乐
func (am *AudioManager) StopMusic() {
	if am.currentMusic != nil {
		am.currentMusic.Pause()
		am.currentMusic = nil
		am.currentMusicID = ""
	}
}

// PauseMusic 暂停当前背景音乐
func (am *AudioManager) PauseMusic() {
	if am.currentMusic != nil {
		am.currentMusic.Pause()
	}
}

// ResumeMusic 恢复当前背景音乐
func (am *AudioManager) ResumeMusic() {
	if am.currentMusic != nil && am.currentMusicID != "" {
		// 检查音乐是否启用
		if am.settingsManager != nil {
			settings := am.settingsManager.GetSettings()
			if !settings.MusicEnabled {
				return
			}
		}
		am.currentMusic.Play()
	}
}

// SetMusicVolume 设置音乐音量
// 此方法立即应用到当前播放的背景音乐
//
// 参数：
//   - volume: 音量值 (0.0 ~ 1.0)
func (am *AudioManager) SetMusicVolume(volume float64) {
	// 更新 SettingsManager
	if am.settingsManager != nil {
		am.settingsManager.SetMusicVolume(volume)
	}

	// 立即应用到当前音乐
	if am.currentMusic != nil {
		am.currentMusic.SetVolume(volume)
	}

	// 更新所有缓存的音乐播放器
	for _, player := range am.musicPlayers {
		player.SetVolume(volume)
	}
}

// SetSoundVolume 设置音效音量
// 此方法会影响后续播放的所有音效
//
// 参数：
//   - volume: 音量值 (0.0 ~ 1.0)
func (am *AudioManager) SetSoundVolume(volume float64) {
	// 更新 SettingsManager
	if am.settingsManager != nil {
		am.settingsManager.SetSoundVolume(volume)
	}

	// 更新所有缓存的音效播放器
	for _, player := range am.soundPlayers {
		player.SetVolume(volume)
	}
}

// GetMusicVolume 获取当前音乐音量
func (am *AudioManager) GetMusicVolume() float64 {
	return am.getMusicVolume()
}

// GetSoundVolume 获取当前音效音量
func (am *AudioManager) GetSoundVolume() float64 {
	return am.getSoundVolume()
}

// getSoundPlayer 获取或加载音效播放器
func (am *AudioManager) getSoundPlayer(soundID string) *audio.Player {
	// 检查缓存
	if player, exists := am.soundPlayers[soundID]; exists {
		log.Printf("[AudioManager] DEBUG: Sound %s found in cache", soundID)
		return player
	}

	// 尝试从 ResourceManager 获取（可能已通过资源ID加载）
	player := am.resourceManager.GetAudioPlayer(soundID)
	if player != nil {
		log.Printf("[AudioManager] DEBUG: Sound %s found in ResourceManager", soundID)
		am.soundPlayers[soundID] = player
		return player
	}

	// 尝试加载（通过资源ID查找路径）
	if am.resourceManager.config != nil {
		if filePath, exists := am.resourceManager.resourceMap[soundID]; exists {
			log.Printf("[AudioManager] DEBUG: Loading sound %s from path: %s", soundID, filePath)
			// 使用 LoadSoundEffect 加载（不循环）
			loadedPlayer, err := am.resourceManager.LoadSoundEffect(filePath)
			if err != nil {
				log.Printf("[AudioManager] Warning: Failed to load sound %s: %v", soundID, err)
				return nil
			}
			am.soundPlayers[soundID] = loadedPlayer
			return loadedPlayer
		}
	}

	log.Printf("[AudioManager] Warning: Sound not found: %s", soundID)
	return nil
}

// getMusicPlayer 获取或加载音乐播放器
func (am *AudioManager) getMusicPlayer(musicID string) *audio.Player {
	// 检查缓存
	if player, exists := am.musicPlayers[musicID]; exists {
		return player
	}

	// 尝试从 ResourceManager 获取
	player := am.resourceManager.GetAudioPlayer(musicID)
	if player != nil {
		am.musicPlayers[musicID] = player
		return player
	}

	// 尝试加载（通过资源ID查找路径）
	if am.resourceManager.config != nil {
		if filePath, exists := am.resourceManager.resourceMap[musicID]; exists {
			// 使用 LoadAudio 加载（循环播放）
			loadedPlayer, err := am.resourceManager.LoadAudio(filePath)
			if err != nil {
				log.Printf("[AudioManager] Warning: Failed to load music %s: %v", musicID, err)
				return nil
			}
			am.musicPlayers[musicID] = loadedPlayer
			return loadedPlayer
		}
	}

	log.Printf("[AudioManager] Warning: Music not found: %s", musicID)
	return nil
}

// getMusicVolume 获取音乐音量设置
func (am *AudioManager) getMusicVolume() float64 {
	if am.settingsManager != nil {
		return am.settingsManager.GetSettings().MusicVolume
	}
	return 0.7 // 默认值
}

// getSoundVolume 获取音效音量设置
func (am *AudioManager) getSoundVolume() float64 {
	if am.settingsManager != nil {
		return am.settingsManager.GetSettings().SoundVolume
	}
	return 0.8 // 默认值
}

// PreloadSounds 预加载音效
// 在场景初始化时调用，避免首次播放时的延迟
//
// 参数：
//   - soundIDs: 要预加载的音效资源ID列表
func (am *AudioManager) PreloadSounds(soundIDs []string) {
	for _, soundID := range soundIDs {
		am.getSoundPlayer(soundID)
	}
	log.Printf("[AudioManager] Preloaded %d sounds", len(soundIDs))
}

// PreloadMusic 预加载背景音乐
// 在场景初始化时调用，避免首次播放时的延迟
//
// 参数：
//   - musicIDs: 要预加载的音乐资源ID列表
func (am *AudioManager) PreloadMusic(musicIDs []string) {
	for _, musicID := range musicIDs {
		am.getMusicPlayer(musicID)
	}
	log.Printf("[AudioManager] Preloaded %d music tracks", len(musicIDs))
}
