package scenes

import (
	"github.com/gonewx/pvz/pkg/game"
)

// Scene is a type alias for game.Scene to maintain backward compatibility.
// All scene implementations should implement the game.Scene interface.
type Scene = game.Scene
