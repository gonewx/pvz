package components

import (
	"testing"
)

func TestNewConveyorBeltComponent(t *testing.T) {
	comp := NewConveyorBeltComponent()

	// 验证默认值
	if comp.Capacity != DefaultConveyorCapacity {
		t.Errorf("Expected capacity %d, got %d", DefaultConveyorCapacity, comp.Capacity)
	}

	if comp.GenerationInterval != DefaultCardGenerationInterval {
		t.Errorf("Expected generation interval %.1f, got %.1f",
			DefaultCardGenerationInterval, comp.GenerationInterval)
	}

	if comp.IsActive {
		t.Error("Expected IsActive to be false by default")
	}

	if comp.SelectedCardIndex != -1 {
		t.Errorf("Expected SelectedCardIndex to be -1, got %d", comp.SelectedCardIndex)
	}

	if len(comp.Cards) != 0 {
		t.Errorf("Expected empty cards slice, got %d cards", len(comp.Cards))
	}
}

func TestConveyorBeltComponent_IsFull(t *testing.T) {
	comp := NewConveyorBeltComponent()
	comp.Capacity = 3

	// 空时不满
	if comp.IsFull() {
		t.Error("Expected IsFull() to be false when empty")
	}

	// 添加卡片
	comp.Cards = append(comp.Cards, ConveyorCard{CardType: CardTypeWallnutBowling})
	comp.Cards = append(comp.Cards, ConveyorCard{CardType: CardTypeWallnutBowling})

	// 未满
	if comp.IsFull() {
		t.Error("Expected IsFull() to be false with 2/3 cards")
	}

	// 满
	comp.Cards = append(comp.Cards, ConveyorCard{CardType: CardTypeExplodeONut})
	if !comp.IsFull() {
		t.Error("Expected IsFull() to be true with 3/3 cards")
	}
}

func TestConveyorBeltComponent_IsEmpty(t *testing.T) {
	comp := NewConveyorBeltComponent()

	// 空时为空
	if !comp.IsEmpty() {
		t.Error("Expected IsEmpty() to be true when empty")
	}

	// 添加卡片后不为空
	comp.Cards = append(comp.Cards, ConveyorCard{CardType: CardTypeWallnutBowling})
	if comp.IsEmpty() {
		t.Error("Expected IsEmpty() to be false with cards")
	}
}

func TestConveyorBeltComponent_CardCount(t *testing.T) {
	comp := NewConveyorBeltComponent()

	if comp.CardCount() != 0 {
		t.Errorf("Expected CardCount() to be 0, got %d", comp.CardCount())
	}

	comp.Cards = append(comp.Cards, ConveyorCard{CardType: CardTypeWallnutBowling})
	comp.Cards = append(comp.Cards, ConveyorCard{CardType: CardTypeExplodeONut})

	if comp.CardCount() != 2 {
		t.Errorf("Expected CardCount() to be 2, got %d", comp.CardCount())
	}
}

func TestCardTypeConstants(t *testing.T) {
	// 验证常量值
	if CardTypeWallnutBowling != "wallnut_bowling" {
		t.Errorf("Expected CardTypeWallnutBowling to be 'wallnut_bowling', got '%s'", CardTypeWallnutBowling)
	}

	if CardTypeExplodeONut != "explode_o_nut" {
		t.Errorf("Expected CardTypeExplodeONut to be 'explode_o_nut', got '%s'", CardTypeExplodeONut)
	}
}

func TestConveyorCard(t *testing.T) {
	card := ConveyorCard{
		CardType:      CardTypeWallnutBowling,
		SlideProgress: 0.5,
		SlotIndex:     2,
	}

	if card.CardType != CardTypeWallnutBowling {
		t.Errorf("Expected CardType '%s', got '%s'", CardTypeWallnutBowling, card.CardType)
	}

	if card.SlideProgress != 0.5 {
		t.Errorf("Expected SlideProgress 0.5, got %f", card.SlideProgress)
	}

	if card.SlotIndex != 2 {
		t.Errorf("Expected SlotIndex 2, got %d", card.SlotIndex)
	}
}
