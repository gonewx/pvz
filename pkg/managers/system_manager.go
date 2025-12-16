// Package managers provides high-level management functionality for the game.
// SystemManager provides unified ECS system management to avoid code duplication
// between GameScene and verify_gameplay tools.
package managers

import (
	"log"

	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/gonewx/pvz/pkg/systems/behavior"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// SystemDependencies contains external dependencies required by systems.
// These are resources and managers that systems need but don't own.
type SystemDependencies struct {
	EntityManager   *ecs.EntityManager
	ResourceManager *game.ResourceManager
	GameState       *game.GameState
	SceneManager    *game.SceneManager // Optional, used by RewardSystem

	// Grid and positioning
	LawnGridEntityID ecs.EntityID
	EnabledLanes     []int

	// Sun collection target position
	SunCollectionTargetX float64
	SunCollectionTargetY float64

	// Sun spawn area bounds
	SunSpawnMinX       float64
	SunSpawnMaxX       float64
	SunSpawnMinTargetY float64
	SunSpawnMaxTargetY float64

	// Level configuration (required for WaveSpawnSystem, TutorialSystem, etc.)
	LevelConfig *config.LevelConfig

	// Optional spawn rules and physics config
	SpawnRulesConfig    *config.SpawnRulesConfig
	ZombiePhysicsConfig *config.ZombiePhysicsConfig

	// Fonts for UI systems
	TitleFont   *text.GoTextFace
	MessageFont *text.GoTextFace
	ButtonFont  *text.GoTextFace

	// Window dimensions for dialog systems
	WindowWidth  int
	WindowHeight int
}

// SystemOptions controls which optional systems should be created.
// By default, all core systems are created. Optional systems are created
// based on these flags and level configuration.
type SystemOptions struct {
	// Tutorial systems (based on level config)
	EnableTutorial       bool
	EnableGuidedTutorial bool

	// Animation systems (based on level config)
	EnableOpeningAnimation bool
	EnableReadySetPlant    bool
	EnableSodding          bool

	// Conveyor belt (Level 1-5, 1-10 etc.)
	EnableConveyorBelt bool

	// Bowling (Level 1-5)
	EnableBowlingNut bool

	// Dave dialogue (Level 1-5)
	EnableDaveDialogue bool

	// Camera system (needed for opening animation)
	EnableCamera bool

	// Zombie groan sounds
	EnableZombieGroan bool

	// Input system (needed for player interaction)
	EnableInput bool

	// UI systems
	EnableButton      bool
	EnableSlider      bool
	EnableCheckbox    bool
	EnableDialog      bool
	EnableProgressBar bool

	// Plant preview (drag and drop planting)
	EnablePlantPreview bool

	// Shovel interaction
	EnableShovelInteraction bool
}

// DefaultSystemOptions returns a SystemOptions with all core systems enabled.
// Use this for standard gameplay scenes.
func DefaultSystemOptions() SystemOptions {
	return SystemOptions{
		EnableTutorial:          false, // Depends on level config
		EnableGuidedTutorial:    false, // Depends on level config
		EnableOpeningAnimation:  false, // Depends on level config
		EnableReadySetPlant:     false, // Depends on level config
		EnableSodding:           false, // Depends on level config
		EnableConveyorBelt:      false, // Depends on level config
		EnableBowlingNut:        false, // Depends on level config
		EnableDaveDialogue:      false, // Depends on level config
		EnableCamera:            true,  // Usually needed
		EnableZombieGroan:       true,  // Environmental sound
		EnableInput:             true,  // Player interaction
		EnableButton:            true,  // UI
		EnableSlider:            true,  // UI
		EnableCheckbox:          true,  // UI
		EnableDialog:            true,  // UI
		EnableProgressBar:       true,  // UI
		EnablePlantPreview:      true,  // Drag and drop
		EnableShovelInteraction: true,  // Shovel tool
	}
}

// VerifyGameplayOptions returns SystemOptions configured for verify_gameplay tool.
// This enables a minimal set of systems for testing.
func VerifyGameplayOptions() SystemOptions {
	return SystemOptions{
		EnableTutorial:          false,
		EnableGuidedTutorial:    false,
		EnableOpeningAnimation:  false,
		EnableReadySetPlant:     false,
		EnableSodding:           false,
		EnableConveyorBelt:      false,
		EnableBowlingNut:        false,
		EnableDaveDialogue:      false,
		EnableCamera:            false, // Not needed for verify
		EnableZombieGroan:       false, // Not needed for verify
		EnableInput:             true,  // Use standard InputSystem for planting logic
		EnableButton:            false, // Not needed for verify
		EnableSlider:            false, // Not needed for verify
		EnableCheckbox:          false, // Not needed for verify
		EnableDialog:            false, // Not needed for verify
		EnableProgressBar:       false, // Not needed for verify
		EnablePlantPreview:      true,  // Useful for testing
		EnableShovelInteraction: false, // Not needed for verify
	}
}

// SystemManager manages all ECS systems used in the game.
// It handles system creation with correct dependency order and provides
// an Update method that calls all systems in the correct execution order.
type SystemManager struct {
	// External dependencies
	deps SystemDependencies
	opts SystemOptions

	// ========================================
	// Core Systems (always created)
	// ========================================

	// Animation and rendering
	reanimSystem *systems.ReanimSystem
	renderSystem *systems.RenderSystem

	// Grid management
	lawnGridSystem *systems.LawnGridSystem

	// Entity behavior and physics
	behaviorSystem *behavior.BehaviorSystem
	physicsSystem  *systems.PhysicsSystem

	// Special movement systems
	poleVaultSystem  *systems.PoleVaultSystem
	slowEffectSystem *systems.SlowEffectSystem

	// Effects
	particleSystem    *systems.ParticleSystem
	flashEffectSystem *systems.FlashEffectSystem

	// Entity lifecycle
	lifetimeSystem *systems.LifetimeSystem

	// ========================================
	// Game Logic Systems (always created)
	// ========================================

	// Sun mechanics
	sunSpawnSystem      *systems.SunSpawnSystem
	sunMovementSystem   *systems.SunMovementSystem
	sunCollectionSystem *systems.SunCollectionSystem

	// Wave and level management
	waveSpawnSystem *systems.WaveSpawnSystem
	levelSystem     *systems.LevelSystem

	// Zombie lane transitions
	zombieLaneTransitionSystem *systems.ZombieLaneTransitionSystem

	// Defense
	lawnmowerSystem *systems.LawnmowerSystem

	// Game state transitions
	finalWaveWarningSystem *systems.FinalWaveWarningSystem
	zombiesWonPhaseSystem  *systems.ZombiesWonPhaseSystem

	// Rewards
	rewardSystem *systems.RewardAnimationSystem

	// ========================================
	// UI Systems (conditionally created)
	// ========================================

	inputSystem                  *systems.InputSystem
	buttonSystem                 *systems.ButtonSystem
	buttonRenderSystem           *systems.ButtonRenderSystem
	sliderSystem                 *systems.SliderSystem
	checkboxSystem               *systems.CheckboxSystem
	dialogInputSystem            *systems.DialogInputSystem
	dialogRenderSystem           *systems.DialogRenderSystem
	levelProgressBarRenderSystem *systems.LevelProgressBarRenderSystem
	plantPreviewSystem           *systems.PlantPreviewSystem
	plantPreviewRenderSystem     *systems.PlantPreviewRenderSystem
	shovelInteractionSystem      *systems.ShovelInteractionSystem

	// ========================================
	// Optional Systems (conditionally created)
	// ========================================

	tutorialSystem       *systems.TutorialSystem
	guidedTutorialSystem *systems.GuidedTutorialSystem
	openingSystem        *systems.OpeningAnimationSystem
	readySetPlantSystem  *systems.ReadySetPlantSystem
	soddingSystem        *systems.SoddingSystem
	cameraSystem         *systems.CameraSystem
	conveyorBeltSystem   *systems.ConveyorBeltSystem
	bowlingNutSystem     *systems.BowlingNutSystem
	daveDialogueSystem   *systems.DaveDialogueSystem
	zombieGroanSystem    *systems.ZombieGroanSystem
	levelPhaseSystem     *systems.LevelPhaseSystem
}

// NewSystemManager creates a new SystemManager with all systems initialized
// in the correct dependency order.
func NewSystemManager(deps SystemDependencies, opts SystemOptions) *SystemManager {
	sm := &SystemManager{
		deps: deps,
		opts: opts,
	}

	// Create systems in dependency order
	sm.createCoreSystems()
	sm.createGameLogicSystems()
	sm.createUISystems()
	sm.createOptionalSystems()

	log.Printf("[SystemManager] All systems initialized")
	return sm
}

// createCoreSystems creates the fundamental systems that everything depends on.
func (sm *SystemManager) createCoreSystems() {
	em := sm.deps.EntityManager
	rm := sm.deps.ResourceManager

	// 1. ReanimSystem (no dependencies, other systems depend on it)
	sm.reanimSystem = systems.NewReanimSystem(em)
	if configManager := rm.GetReanimConfigManager(); configManager != nil {
		sm.reanimSystem.SetConfigManager(configManager)
	}
	sm.reanimSystem.SetResourceLoader(rm)
	log.Printf("[SystemManager] Created ReanimSystem")

	// 2. RenderSystem (depends on ReanimSystem)
	sm.renderSystem = systems.NewRenderSystem(em)
	sm.renderSystem.SetReanimSystem(sm.reanimSystem)
	sm.renderSystem.SetResourceManager(rm)
	log.Printf("[SystemManager] Created RenderSystem")

	// 3. LawnGridSystem (no dependencies)
	sm.lawnGridSystem = systems.NewLawnGridSystem(em, sm.deps.EnabledLanes)
	log.Printf("[SystemManager] Created LawnGridSystem with lanes: %v", sm.deps.EnabledLanes)

	// 4. BehaviorSystem (depends on LawnGridSystem)
	sm.behaviorSystem = behavior.NewBehaviorSystem(
		em, rm, sm.deps.GameState,
		sm.lawnGridSystem, sm.deps.LawnGridEntityID,
		sm.reanimSystem,
	)
	log.Printf("[SystemManager] Created BehaviorSystem")

	// 5. PhysicsSystem (no dependencies)
	sm.physicsSystem = systems.NewPhysicsSystem(em, rm)
	log.Printf("[SystemManager] Created PhysicsSystem")

	// 6. PoleVaultSystem (requires GameState for sound effects)
	sm.poleVaultSystem = systems.NewPoleVaultSystem(em, sm.deps.GameState)
	log.Printf("[SystemManager] Created PoleVaultSystem")

	// 7. SlowEffectSystem (no dependencies)
	sm.slowEffectSystem = systems.NewSlowEffectSystem(em)
	log.Printf("[SystemManager] Created SlowEffectSystem")

	// 8. ParticleSystem (needs ResourceManager)
	sm.particleSystem = systems.NewParticleSystem(em, rm)
	log.Printf("[SystemManager] Created ParticleSystem")

	// 9. FlashEffectSystem (no dependencies)
	sm.flashEffectSystem = systems.NewFlashEffectSystem(em)
	log.Printf("[SystemManager] Created FlashEffectSystem")

	// 10. LifetimeSystem (no dependencies, but runs last)
	sm.lifetimeSystem = systems.NewLifetimeSystem(em)
	log.Printf("[SystemManager] Created LifetimeSystem")
}

// createGameLogicSystems creates the game logic systems.
func (sm *SystemManager) createGameLogicSystems() {
	em := sm.deps.EntityManager
	rm := sm.deps.ResourceManager
	gs := sm.deps.GameState

	// 1. Sun systems
	sm.sunSpawnSystem = systems.NewSunSpawnSystem(
		em, rm,
		sm.deps.SunSpawnMinX, sm.deps.SunSpawnMaxX,
		sm.deps.SunSpawnMinTargetY, sm.deps.SunSpawnMaxTargetY,
	)
	sm.sunMovementSystem = systems.NewSunMovementSystem(em)
	sm.sunCollectionSystem = systems.NewSunCollectionSystem(
		em, gs,
		sm.deps.SunCollectionTargetX, sm.deps.SunCollectionTargetY,
	)
	log.Printf("[SystemManager] Created Sun systems")

	// 2. WaveSpawnSystem
	sm.waveSpawnSystem = systems.NewWaveSpawnSystem(
		em, rm,
		sm.deps.LevelConfig, gs,
		sm.deps.SpawnRulesConfig, sm.deps.ZombiePhysicsConfig,
	)
	log.Printf("[SystemManager] Created WaveSpawnSystem")

	// 3. LawnmowerSystem
	sm.lawnmowerSystem = systems.NewLawnmowerSystem(em, rm, gs)
	log.Printf("[SystemManager] Created LawnmowerSystem")

	// 4. RewardAnimationSystem (depends on ReanimSystem, ParticleSystem, RenderSystem)
	sm.rewardSystem = systems.NewRewardAnimationSystem(
		em, gs, rm, sm.deps.SceneManager,
		sm.reanimSystem, sm.particleSystem, sm.renderSystem,
	)
	log.Printf("[SystemManager] Created RewardAnimationSystem")

	// 5. LevelSystem (depends on WaveSpawnSystem, RewardSystem, LawnmowerSystem)
	sm.levelSystem = systems.NewLevelSystem(
		em, gs, sm.waveSpawnSystem, rm,
		sm.rewardSystem, sm.lawnmowerSystem,
	)
	if sm.deps.ZombiePhysicsConfig != nil {
		sm.levelSystem.SetZombiePhysicsConfig(sm.deps.ZombiePhysicsConfig)
	}
	log.Printf("[SystemManager] Created LevelSystem")

	// 6. ZombieLaneTransitionSystem
	sm.zombieLaneTransitionSystem = systems.NewZombieLaneTransitionSystem(em)
	log.Printf("[SystemManager] Created ZombieLaneTransitionSystem")

	// 7. FinalWaveWarningSystem
	sm.finalWaveWarningSystem = systems.NewFinalWaveWarningSystem(em)
	log.Printf("[SystemManager] Created FinalWaveWarningSystem")

	// 8. ZombiesWonPhaseSystem
	sm.zombiesWonPhaseSystem = systems.NewZombiesWonPhaseSystem(
		em, rm, gs,
		sm.deps.WindowWidth, sm.deps.WindowHeight,
	)
	log.Printf("[SystemManager] Created ZombiesWonPhaseSystem")
}

// createUISystems creates UI-related systems based on options.
func (sm *SystemManager) createUISystems() {
	em := sm.deps.EntityManager
	rm := sm.deps.ResourceManager
	gs := sm.deps.GameState

	// InputSystem
	if sm.opts.EnableInput {
		sm.inputSystem = systems.NewInputSystem(
			em, rm, gs, sm.reanimSystem,
			sm.deps.SunCollectionTargetX, sm.deps.SunCollectionTargetY,
			sm.lawnGridSystem, sm.deps.LawnGridEntityID,
		)
		log.Printf("[SystemManager] Created InputSystem")
	}

	// Button systems
	if sm.opts.EnableButton {
		sm.buttonSystem = systems.NewButtonSystem(em)
		sm.buttonRenderSystem = systems.NewButtonRenderSystem(em)
		log.Printf("[SystemManager] Created Button systems")
	}

	// Slider system
	if sm.opts.EnableSlider {
		sm.sliderSystem = systems.NewSliderSystem(em)
		log.Printf("[SystemManager] Created SliderSystem")
	}

	// Checkbox system
	if sm.opts.EnableCheckbox {
		sm.checkboxSystem = systems.NewCheckboxSystem(em)
		log.Printf("[SystemManager] Created CheckboxSystem")
	}

	// Dialog systems
	if sm.opts.EnableDialog {
		sm.dialogInputSystem = systems.NewDialogInputSystem(em)
		sm.dialogRenderSystem = systems.NewDialogRenderSystem(
			em, sm.deps.WindowWidth, sm.deps.WindowHeight,
			sm.deps.TitleFont, sm.deps.MessageFont, sm.deps.ButtonFont,
		)
		log.Printf("[SystemManager] Created Dialog systems")
	}

	// Plant preview systems
	if sm.opts.EnablePlantPreview {
		sm.plantPreviewSystem = systems.NewPlantPreviewSystem(em, gs, sm.lawnGridSystem)
		sm.plantPreviewSystem.SetLawnGridEntityID(sm.deps.LawnGridEntityID)
		sm.plantPreviewRenderSystem = systems.NewPlantPreviewRenderSystem(em, sm.plantPreviewSystem)
		log.Printf("[SystemManager] Created PlantPreview systems")
	}

	// Shovel interaction
	if sm.opts.EnableShovelInteraction {
		sm.shovelInteractionSystem = systems.NewShovelInteractionSystem(em, gs, rm)
		log.Printf("[SystemManager] Created ShovelInteractionSystem")
	}
}

// createOptionalSystems creates optional systems based on options.
func (sm *SystemManager) createOptionalSystems() {
	em := sm.deps.EntityManager
	rm := sm.deps.ResourceManager
	gs := sm.deps.GameState

	// Camera system (needed for opening animation)
	if sm.opts.EnableCamera {
		sm.cameraSystem = systems.NewCameraSystem(em, gs)
		log.Printf("[SystemManager] Created CameraSystem")
	}

	// Opening animation (depends on CameraSystem)
	if sm.opts.EnableOpeningAnimation && sm.deps.LevelConfig != nil {
		sm.openingSystem = systems.NewOpeningAnimationSystem(
			em, gs, rm, sm.deps.LevelConfig, sm.cameraSystem,
		)
		if sm.openingSystem != nil {
			log.Printf("[SystemManager] Created OpeningAnimationSystem")
		}
	}

	// ReadySetPlant
	if sm.opts.EnableReadySetPlant {
		sm.readySetPlantSystem = systems.NewReadySetPlantSystem(em, rm)
		log.Printf("[SystemManager] Created ReadySetPlantSystem")
	}

	// Sodding
	if sm.opts.EnableSodding {
		sm.soddingSystem = systems.NewSoddingSystem(em, rm)
		log.Printf("[SystemManager] Created SoddingSystem")
	}

	// Tutorial
	if sm.opts.EnableTutorial && sm.deps.LevelConfig != nil {
		sm.tutorialSystem = systems.NewTutorialSystem(
			em, gs, rm,
			sm.lawnGridSystem, sm.sunSpawnSystem,
			sm.deps.LevelConfig,
		)
		sm.tutorialSystem.SetLevelSystem(sm.levelSystem)
		log.Printf("[SystemManager] Created TutorialSystem")
	}

	// Guided tutorial
	if sm.opts.EnableGuidedTutorial {
		sm.guidedTutorialSystem = systems.NewGuidedTutorialSystem(em, gs, rm)
		log.Printf("[SystemManager] Created GuidedTutorialSystem")
	}

	// Level phase
	if sm.opts.EnableGuidedTutorial || sm.opts.EnableConveyorBelt {
		sm.levelPhaseSystem = systems.NewLevelPhaseSystem(em, gs, rm)
		log.Printf("[SystemManager] Created LevelPhaseSystem")
	}

	// Conveyor belt
	if sm.opts.EnableConveyorBelt {
		sm.conveyorBeltSystem = systems.NewConveyorBeltSystem(em, gs, rm)
		log.Printf("[SystemManager] Created ConveyorBeltSystem")
	}

	// Bowling nut
	if sm.opts.EnableBowlingNut {
		sm.bowlingNutSystem = systems.NewBowlingNutSystem(em, rm)
		log.Printf("[SystemManager] Created BowlingNutSystem")
	}

	// Dave dialogue
	if sm.opts.EnableDaveDialogue {
		sm.daveDialogueSystem = systems.NewDaveDialogueSystem(em, gs, rm)
		log.Printf("[SystemManager] Created DaveDialogueSystem")
	}

	// Zombie groan
	if sm.opts.EnableZombieGroan {
		sm.zombieGroanSystem = systems.NewZombieGroanSystem(em, gs)
		log.Printf("[SystemManager] Created ZombieGroanSystem")
	}
}

// Update updates all systems in the correct execution order.
// This method respects the critical ordering constraints documented in the story.
func (sm *SystemManager) Update(deltaTime float64) {
	// ========================================
	// Phase 1: Animation Processing
	// ========================================
	// ReanimSystem must run first to process animation commands
	// before BehaviorSystem reads animation state
	sm.reanimSystem.Update(deltaTime)

	// ========================================
	// Phase 2: Pre-Behavior Systems
	// ========================================
	// PoleVaultSystem must run before BehaviorSystem
	// (checks for plant collision and triggers jump)
	if sm.poleVaultSystem != nil {
		sm.poleVaultSystem.Update(deltaTime)
	}

	// ========================================
	// Phase 3: Core Game Logic
	// ========================================
	// Level and wave management
	if sm.levelSystem != nil {
		sm.levelSystem.Update(deltaTime)
	}
	if sm.waveSpawnSystem != nil {
		sm.waveSpawnSystem.UpdatePendingActivations(deltaTime)
	}

	// Zombie lane transitions (before behavior)
	if sm.zombieLaneTransitionSystem != nil {
		sm.zombieLaneTransitionSystem.Update(deltaTime)
	}

	// Entity behaviors (plants, zombies, projectiles)
	sm.behaviorSystem.Update(deltaTime)

	// Physics and collision
	sm.physicsSystem.Update(deltaTime)

	// SlowEffectSystem must run after PhysicsSystem
	// (processes slow effect durations after damage is dealt)
	if sm.slowEffectSystem != nil {
		sm.slowEffectSystem.Update(deltaTime)
	}

	// Lawnmower activation and movement
	if sm.lawnmowerSystem != nil {
		sm.lawnmowerSystem.Update(deltaTime)
	}

	// ========================================
	// Phase 4: Sun Mechanics
	// ========================================
	if sm.sunSpawnSystem != nil {
		sm.sunSpawnSystem.Update(deltaTime)
	}
	if sm.sunMovementSystem != nil {
		sm.sunMovementSystem.Update(deltaTime)
	}
	if sm.sunCollectionSystem != nil {
		sm.sunCollectionSystem.Update(deltaTime)
	}

	// ========================================
	// Phase 5: Visual Effects
	// ========================================
	sm.particleSystem.Update(deltaTime)
	if sm.flashEffectSystem != nil {
		sm.flashEffectSystem.Update(deltaTime)
	}

	// ========================================
	// Phase 6: Game State Transitions
	// ========================================
	if sm.rewardSystem != nil {
		sm.rewardSystem.Update(deltaTime)
	}
	if sm.finalWaveWarningSystem != nil {
		sm.finalWaveWarningSystem.Update(deltaTime)
	}
	if sm.zombiesWonPhaseSystem != nil {
		sm.zombiesWonPhaseSystem.Update(deltaTime)
	}

	// ========================================
	// Phase 7: Optional Game Systems
	// ========================================
	if sm.cameraSystem != nil {
		sm.cameraSystem.Update(deltaTime)
	}
	if sm.openingSystem != nil {
		sm.openingSystem.Update(deltaTime)
	}
	if sm.readySetPlantSystem != nil {
		sm.readySetPlantSystem.Update(deltaTime)
	}
	if sm.soddingSystem != nil {
		sm.soddingSystem.Update(deltaTime)
	}
	if sm.tutorialSystem != nil {
		sm.tutorialSystem.Update(deltaTime)
	}
	if sm.guidedTutorialSystem != nil {
		sm.guidedTutorialSystem.Update(deltaTime)
	}
	if sm.levelPhaseSystem != nil {
		sm.levelPhaseSystem.Update(deltaTime)
	}
	if sm.conveyorBeltSystem != nil {
		sm.conveyorBeltSystem.Update(deltaTime)
	}
	if sm.bowlingNutSystem != nil {
		sm.bowlingNutSystem.Update(deltaTime)
	}
	if sm.daveDialogueSystem != nil {
		sm.daveDialogueSystem.Update(deltaTime)
	}
	if sm.zombieGroanSystem != nil {
		sm.zombieGroanSystem.Update(deltaTime)
	}

	// ========================================
	// Phase 8: UI Systems
	// ========================================
	if sm.plantPreviewSystem != nil {
		sm.plantPreviewSystem.Update(deltaTime)
	}
	if sm.lawnGridSystem != nil {
		sm.lawnGridSystem.Update(deltaTime)
	}
	if sm.buttonSystem != nil {
		sm.buttonSystem.Update(deltaTime)
	}
	if sm.sliderSystem != nil {
		sm.sliderSystem.Update(deltaTime)
	}
	if sm.checkboxSystem != nil {
		sm.checkboxSystem.Update(deltaTime)
	}
	if sm.dialogInputSystem != nil {
		sm.dialogInputSystem.Update(deltaTime)
	}

	// ========================================
	// Phase 9: Cleanup (always last)
	// ========================================
	sm.lifetimeSystem.Update(deltaTime)
	sm.deps.EntityManager.RemoveMarkedEntities()
}

// ========================================
// Getter Methods for System Access
// ========================================

// GetReanimSystem returns the Reanim animation system.
func (sm *SystemManager) GetReanimSystem() *systems.ReanimSystem {
	return sm.reanimSystem
}

// GetRenderSystem returns the render system.
func (sm *SystemManager) GetRenderSystem() *systems.RenderSystem {
	return sm.renderSystem
}

// GetLawnGridSystem returns the lawn grid system.
func (sm *SystemManager) GetLawnGridSystem() *systems.LawnGridSystem {
	return sm.lawnGridSystem
}

// GetBehaviorSystem returns the behavior system.
func (sm *SystemManager) GetBehaviorSystem() *behavior.BehaviorSystem {
	return sm.behaviorSystem
}

// GetPhysicsSystem returns the physics system.
func (sm *SystemManager) GetPhysicsSystem() *systems.PhysicsSystem {
	return sm.physicsSystem
}

// GetPoleVaultSystem returns the pole vault system.
func (sm *SystemManager) GetPoleVaultSystem() *systems.PoleVaultSystem {
	return sm.poleVaultSystem
}

// GetSlowEffectSystem returns the slow effect system.
func (sm *SystemManager) GetSlowEffectSystem() *systems.SlowEffectSystem {
	return sm.slowEffectSystem
}

// GetParticleSystem returns the particle system.
func (sm *SystemManager) GetParticleSystem() *systems.ParticleSystem {
	return sm.particleSystem
}

// GetFlashEffectSystem returns the flash effect system.
func (sm *SystemManager) GetFlashEffectSystem() *systems.FlashEffectSystem {
	return sm.flashEffectSystem
}

// GetLifetimeSystem returns the lifetime system.
func (sm *SystemManager) GetLifetimeSystem() *systems.LifetimeSystem {
	return sm.lifetimeSystem
}

// GetSunSpawnSystem returns the sun spawn system.
func (sm *SystemManager) GetSunSpawnSystem() *systems.SunSpawnSystem {
	return sm.sunSpawnSystem
}

// GetSunMovementSystem returns the sun movement system.
func (sm *SystemManager) GetSunMovementSystem() *systems.SunMovementSystem {
	return sm.sunMovementSystem
}

// GetSunCollectionSystem returns the sun collection system.
func (sm *SystemManager) GetSunCollectionSystem() *systems.SunCollectionSystem {
	return sm.sunCollectionSystem
}

// GetWaveSpawnSystem returns the wave spawn system.
func (sm *SystemManager) GetWaveSpawnSystem() *systems.WaveSpawnSystem {
	return sm.waveSpawnSystem
}

// GetLevelSystem returns the level system.
func (sm *SystemManager) GetLevelSystem() *systems.LevelSystem {
	return sm.levelSystem
}

// GetZombieLaneTransitionSystem returns the zombie lane transition system.
func (sm *SystemManager) GetZombieLaneTransitionSystem() *systems.ZombieLaneTransitionSystem {
	return sm.zombieLaneTransitionSystem
}

// GetLawnmowerSystem returns the lawnmower system.
func (sm *SystemManager) GetLawnmowerSystem() *systems.LawnmowerSystem {
	return sm.lawnmowerSystem
}

// GetFinalWaveWarningSystem returns the final wave warning system.
func (sm *SystemManager) GetFinalWaveWarningSystem() *systems.FinalWaveWarningSystem {
	return sm.finalWaveWarningSystem
}

// GetZombiesWonPhaseSystem returns the zombies won phase system.
func (sm *SystemManager) GetZombiesWonPhaseSystem() *systems.ZombiesWonPhaseSystem {
	return sm.zombiesWonPhaseSystem
}

// GetRewardSystem returns the reward animation system.
func (sm *SystemManager) GetRewardSystem() *systems.RewardAnimationSystem {
	return sm.rewardSystem
}

// GetInputSystem returns the input system.
func (sm *SystemManager) GetInputSystem() *systems.InputSystem {
	return sm.inputSystem
}

// GetButtonSystem returns the button system.
func (sm *SystemManager) GetButtonSystem() *systems.ButtonSystem {
	return sm.buttonSystem
}

// GetButtonRenderSystem returns the button render system.
func (sm *SystemManager) GetButtonRenderSystem() *systems.ButtonRenderSystem {
	return sm.buttonRenderSystem
}

// GetSliderSystem returns the slider system.
func (sm *SystemManager) GetSliderSystem() *systems.SliderSystem {
	return sm.sliderSystem
}

// GetCheckboxSystem returns the checkbox system.
func (sm *SystemManager) GetCheckboxSystem() *systems.CheckboxSystem {
	return sm.checkboxSystem
}

// GetDialogInputSystem returns the dialog input system.
func (sm *SystemManager) GetDialogInputSystem() *systems.DialogInputSystem {
	return sm.dialogInputSystem
}

// GetDialogRenderSystem returns the dialog render system.
func (sm *SystemManager) GetDialogRenderSystem() *systems.DialogRenderSystem {
	return sm.dialogRenderSystem
}

// GetLevelProgressBarRenderSystem returns the level progress bar render system.
func (sm *SystemManager) GetLevelProgressBarRenderSystem() *systems.LevelProgressBarRenderSystem {
	return sm.levelProgressBarRenderSystem
}

// GetPlantPreviewSystem returns the plant preview system.
func (sm *SystemManager) GetPlantPreviewSystem() *systems.PlantPreviewSystem {
	return sm.plantPreviewSystem
}

// GetPlantPreviewRenderSystem returns the plant preview render system.
func (sm *SystemManager) GetPlantPreviewRenderSystem() *systems.PlantPreviewRenderSystem {
	return sm.plantPreviewRenderSystem
}

// GetShovelInteractionSystem returns the shovel interaction system.
func (sm *SystemManager) GetShovelInteractionSystem() *systems.ShovelInteractionSystem {
	return sm.shovelInteractionSystem
}

// GetTutorialSystem returns the tutorial system.
func (sm *SystemManager) GetTutorialSystem() *systems.TutorialSystem {
	return sm.tutorialSystem
}

// GetGuidedTutorialSystem returns the guided tutorial system.
func (sm *SystemManager) GetGuidedTutorialSystem() *systems.GuidedTutorialSystem {
	return sm.guidedTutorialSystem
}

// GetOpeningAnimationSystem returns the opening animation system.
func (sm *SystemManager) GetOpeningAnimationSystem() *systems.OpeningAnimationSystem {
	return sm.openingSystem
}

// GetReadySetPlantSystem returns the ready set plant system.
func (sm *SystemManager) GetReadySetPlantSystem() *systems.ReadySetPlantSystem {
	return sm.readySetPlantSystem
}

// GetSoddingSystem returns the sodding system.
func (sm *SystemManager) GetSoddingSystem() *systems.SoddingSystem {
	return sm.soddingSystem
}

// GetCameraSystem returns the camera system.
func (sm *SystemManager) GetCameraSystem() *systems.CameraSystem {
	return sm.cameraSystem
}

// GetConveyorBeltSystem returns the conveyor belt system.
func (sm *SystemManager) GetConveyorBeltSystem() *systems.ConveyorBeltSystem {
	return sm.conveyorBeltSystem
}

// GetBowlingNutSystem returns the bowling nut system.
func (sm *SystemManager) GetBowlingNutSystem() *systems.BowlingNutSystem {
	return sm.bowlingNutSystem
}

// GetDaveDialogueSystem returns the dave dialogue system.
func (sm *SystemManager) GetDaveDialogueSystem() *systems.DaveDialogueSystem {
	return sm.daveDialogueSystem
}

// GetZombieGroanSystem returns the zombie groan system.
func (sm *SystemManager) GetZombieGroanSystem() *systems.ZombieGroanSystem {
	return sm.zombieGroanSystem
}

// GetLevelPhaseSystem returns the level phase system.
func (sm *SystemManager) GetLevelPhaseSystem() *systems.LevelPhaseSystem {
	return sm.levelPhaseSystem
}

// GetEntityManager returns the entity manager.
func (sm *SystemManager) GetEntityManager() *ecs.EntityManager {
	return sm.deps.EntityManager
}
