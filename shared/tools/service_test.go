package tools

import "testing"

func TestServiceListsAndActivatesTools(t *testing.T) {
	service := NewService()

	list := service.List()
	if len(list) == 0 {
		t.Fatal("expected default tools")
	}

	result := service.Activate("tool-entrance")
	if !result.OK || result.Kind != "panel" {
		t.Fatalf("enabled tool did not activate: %#v", result)
	}

	result = service.Activate("gmail")
	if result.OK {
		t.Fatalf("pending connector should not activate as ready: %#v", result)
	}
}
