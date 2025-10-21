# Technical Debt: Story 10.1 - BGM Control

## Summary

**Story**: 10.1 - 暂停菜单系统 (Pause Menu System)  
**Date Created**: 2025-10-21  
**Status**: Pending (Waiting for BGM System Implementation)  
**Priority**: Medium  
**Estimated Effort**: 1-2 hours (once BGM system is available)

## Description

**Acceptance Criteria 6** from Story 10.1 requires BGM volume control during pause/resume:
> "暂停时，背景音乐音量降低或暂停，恢复时音乐恢复正常。"
> (When paused, BGM volume should be reduced or paused; when resumed, music should return to normal.)

This functionality was **not implemented** because the project currently lacks a BGM (Background Music) playback system.

## Current State

### What Works
- ✅ Pause menu displays and hides correctly
- ✅ Game logic systems stop when paused
- ✅ UI systems continue to update when paused
- ✅ ESC key toggles pause/resume
- ✅ Button callbacks trigger correctly (Continue, Restart, Return to Menu)

### What's Missing
- ❌ BGM volume reduction/pause when entering pause menu
- ❌ BGM volume restoration/resume when exiting pause menu

## Dependency

This functionality depends on:
- **BGM System Implementation** (not yet scheduled)
  - `ResourceManager` or `AudioManager` with BGM control APIs
  - Expected methods:
    - `SetBGMVolume(volume float64)` - Set BGM volume (0.0 - 1.0)
    - `PauseBGM()` - Pause BGM playback
    - `ResumeBGM()` - Resume BGM playback

## Implementation Plan (When BGM System is Ready)

### Code Changes Required

**File**: `pkg/modules/pause_menu_module.go`

```go
// Modify Show() method
func (m *PauseMenuModule) Show() {
    m.gameState.SetPaused(true)
    m.wasActive = true
    
    // ADD: Pause or reduce BGM volume
    if m.onPauseMusic != nil {
        m.onPauseMusic()
    }
    
    log.Printf("[PauseMenuModule] Pause menu shown")
}

// Modify Hide() method
func (m *PauseMenuModule) Hide() {
    m.gameState.SetPaused(false)
    m.wasActive = false
    
    // ADD: Resume or restore BGM volume
    if m.onResumeMusic != nil {
        m.onResumeMusic()
    }
    
    log.Printf("[PauseMenuModule] Pause menu hidden")
}
```

**File**: `pkg/scenes/game_scene.go`

```go
// Modify initPauseMenuModule()
func (s *GameScene) initPauseMenuModule() error {
    // ... existing code ...
    
    pauseMenuModule, err := modules.NewPauseMenuModule(
        // ... existing parameters ...
        modules.PauseMenuCallbacks{
            OnContinue: func() { /* existing */ },
            OnRestart:  func() { /* existing */ },
            OnMainMenu: func() { /* existing */ },
            OnPauseMusic: func() {
                // ADD: Pause or reduce BGM volume
                if s.bgmPlayer != nil {
                    s.bgmPlayer.SetVolume(0.5) // Option 1: Reduce to 50%
                    // OR
                    s.bgmPlayer.Pause()         // Option 2: Pause completely
                }
            },
            OnResumeMusic: func() {
                // ADD: Resume or restore BGM volume
                if s.bgmPlayer != nil {
                    s.bgmPlayer.SetVolume(1.0) // Option 1: Restore to 100%
                    // OR
                    s.bgmPlayer.Play()          // Option 2: Resume playback
                }
            },
        },
    )
    
    // ... rest of code ...
}
```

### Testing Plan

1. **Unit Test**: Mock BGM callbacks and verify they're called during Show/Hide
2. **Integration Test**: Verify BGM volume/state changes in GameScene
3. **Manual Test**: 
   - Start game with BGM playing
   - Pause → BGM volume reduces or pauses
   - Resume → BGM volume restores or resumes

### Acceptance Criteria Validation

Once implemented, verify AC 6:
- [ ] Pressing ESC or clicking Menu button reduces/pauses BGM
- [ ] Clicking "Continue" button restores/resumes BGM
- [ ] BGM state persists across multiple pause/resume cycles
- [ ] No audio glitches or pops during volume changes

## Risk Assessment

**Risk**: Low  
**Impact**: Low (cosmetic/UX improvement, not critical functionality)

- Gameplay is fully functional without BGM control
- Users can manually adjust system volume if needed
- No security, performance, or data integrity concerns

## References

- **Story File**: `docs/stories/10.1.story.md`
- **QA Review**: `docs/qa/gates/10.1-pause-menu-system.yml` (Issue FEAT-001)
- **Related Code**:
  - `pkg/modules/pause_menu_module.go` (callbacks already defined)
  - `pkg/game/resource_manager.go` (BGM API will be added here)

## Notes

- The architecture already supports this functionality via callbacks (`OnPauseMusic`, `OnResumeMusic`)
- Callbacks are defined but currently set to `nil` in GameScene
- Once BGM system exists, implementation is straightforward (estimated 1-2 hours)
- Consider adding configuration option: "Pause BGM" vs "Reduce Volume to X%"

## Resolution Criteria

Mark this debt as **resolved** when:
1. BGM system is implemented in the project
2. Pause menu callbacks control BGM volume/state
3. AC 6 tests pass (manual or automated)
4. No regressions in existing pause menu functionality

---

**Last Updated**: 2025-10-21  
**Tracked By**: James (Full Stack Developer)  
**Epic Dependency**: TBD (BGM System Epic)
