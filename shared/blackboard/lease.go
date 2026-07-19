package blackboard

// lease.go — coordinator lease 的「運行期」判定（spec §9）。
// 投影器只輸出 lease 鏈與最強租約；有效期與搶租/續租決策屬於 app 時鐘，
// 集中在這裡，維持投影器的純函數性。

import "time"

// LeaseRenewWindow：過期前多久內允許續租（spec §9：30 秒）。
const LeaseRenewWindow = 30 * time.Second

// LeaseDecision is what an app instance should do about the lease now.
type LeaseDecision string

const (
	// LeaseAcquire: no valid lease exists — take over with epoch+1.
	LeaseAcquire LeaseDecision = "acquire"
	// LeaseRenew: we hold the lease and are inside the renew window.
	LeaseRenew LeaseDecision = "renew"
	// LeaseHold: we hold a valid lease; nothing to do yet.
	LeaseHold LeaseDecision = "hold"
	// LeaseObserve: someone else holds a valid lease.
	LeaseObserve LeaseDecision = "observe"
)

// LeaseValidAt reports whether the lease is unexpired at the given time.
func LeaseValidAt(l *LeaseView, now time.Time) bool {
	if l == nil || l.LeaseUntil == "" {
		return false
	}
	until, err := time.Parse(time.RFC3339, l.LeaseUntil)
	if err != nil {
		return false
	}
	return now.Before(until)
}

// DecideLease returns the action for holder and the epoch to write when the
// action is LeaseAcquire or LeaseRenew.
//
// Rules (spec §9): expired or absent lease → acquire with epoch+1; own
// lease inside the renew window → renew with the SAME epoch (the projector
// breaks epoch ties by later canonical position, so a renewal wins over the
// holder's own older lease without escalating epochs); someone else's valid
// lease → observe.
func DecideLease(l *LeaseView, holder string, now time.Time) (LeaseDecision, int) {
	if !LeaseValidAt(l, now) {
		epoch := 1
		if l != nil {
			epoch = l.Epoch + 1
		}
		return LeaseAcquire, epoch
	}
	if l.Holder != holder {
		return LeaseObserve, 0
	}
	until, _ := time.Parse(time.RFC3339, l.LeaseUntil)
	if until.Sub(now) <= LeaseRenewWindow {
		return LeaseRenew, l.Epoch
	}
	return LeaseHold, l.Epoch
}
