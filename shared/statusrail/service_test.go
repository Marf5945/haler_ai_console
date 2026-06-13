package statusrail

import (
	"testing"
	"time"
)

func TestBlocklistRejectsWorkflowInput(t *testing.T) {
	if !IsBlocked("Review 還有東西要看嗎？") {
		t.Fatal("expected Review input to be blocked")
	}
	if !IsBlocked("幫我執行這個 DAG") {
		t.Fatal("expected command forwarding input to be blocked")
	}
}

func TestNoticeTakesDisplayForTenSecondsThenReleasesLockLater(t *testing.T) {
	s := NewService(t.TempDir(), []string{"Hi 主人"})
	s.AddNotice("review", NoticeReviewPending, "危險操作", PriorityNormal)

	view := s.View()
	if view.Layer != LayerNotice {
		t.Fatalf("expected L2 notice, got %s", view.Layer)
	}
	if view.LockedCount != 1 {
		t.Fatalf("expected locked notice, got %d", view.LockedCount)
	}

	s.mu.Lock()
	s.notices.active.DisplayUntil = time.Now().Add(-time.Second)
	released := s.viewLocked(time.Now())
	s.mu.Unlock()

	if released.Layer == LayerNotice {
		t.Fatal("expected visual focus to release after display window")
	}
	if released.LockedCount != 1 {
		t.Fatalf("expected safety lock to remain after visual release, got %d", released.LockedCount)
	}
}

func TestStatusRailCompanionRejectsBlockedInput(t *testing.T) {
	s := NewService(t.TempDir(), []string{"Hi 主人"})
	reply := s.CompanionReply("把這段寫入 memory")
	if reply != RejectTemplate {
		t.Fatalf("expected reject template, got %q", reply)
	}
}
