package controlseal

import "testing"

func TestGenerateSealUsesThreeAllowedConsonants(t *testing.T) {
	seal := GenerateSeal()
	runes := []rune(seal)
	if len(runes) != SealLength {
		t.Fatalf("seal length = %d, want %d", len(runes), SealLength)
	}
	if !LooksLikeSeal(seal) {
		t.Fatalf("seal %q should use allowed consonants only", seal)
	}
}

func TestRotateIfNeededAfterNSuccessfulTurns(t *testing.T) {
	manager := NewManager(DefaultSettings())
	initial := manager.CurrentSeal()
	settings := Settings{RotateEverySuccessfulTurns: 3}

	if manager.RotateIfNeeded(2, settings) {
		t.Fatal("should not rotate before N successful turns")
	}
	if manager.CurrentSeal() != initial {
		t.Fatal("seal changed before rotation threshold")
	}
	if !manager.RotateIfNeeded(3, settings) {
		t.Fatal("should rotate at N successful turns")
	}
	if manager.CurrentSeal() == initial {
		t.Fatal("seal should change after rotation")
	}
	if manager.RotateIfNeeded(3, settings) {
		t.Fatal("same successful turn count should not rotate twice")
	}
}

func TestStampCommandDoesNotMutateRawText(t *testing.T) {
	manager := NewManager(DefaultSettings())
	raw := "查台北天氣"
	stamped := manager.StampCommand(raw)
	if stamped == raw {
		t.Fatal("stamped command should include seal")
	}
	if raw != "查台北天氣" {
		t.Fatal("raw user text should remain unchanged")
	}
}
