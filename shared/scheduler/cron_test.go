package scheduler

import (
	"testing"
	"time"
)

func TestCronParseBasicAndMatches(t *testing.T) {
	expr, err := ParseCron("30 9 * * *")
	if err != nil {
		t.Fatalf("ParseCron returned error: %v", err)
	}

	matched := time.Date(2026, 5, 21, 9, 30, 45, 0, time.UTC)
	if !expr.Matches(matched) {
		t.Fatalf("expected 09:30 to match")
	}
	notMatched := time.Date(2026, 5, 21, 9, 31, 0, 0, time.UTC)
	if expr.Matches(notMatched) {
		t.Fatalf("expected 09:31 not to match")
	}
}

func TestCronStepRangeListAndShortcut(t *testing.T) {
	step, err := ParseCron("*/15 * * * *")
	if err != nil {
		t.Fatalf("ParseCron step returned error: %v", err)
	}
	for _, minute := range []int{0, 15, 30, 45} {
		if !contains(step.Minute, minute) {
			t.Fatalf("expected minute %d in step expression", minute)
		}
	}

	ranged, err := ParseCron("0 9-17 * * 1,3,5")
	if err != nil {
		t.Fatalf("ParseCron range/list returned error: %v", err)
	}
	if !ranged.Matches(time.Date(2026, 5, 22, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected Friday 09:00 to match")
	}
	if ranged.Matches(time.Date(2026, 5, 24, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected Sunday 09:00 not to match")
	}

	daily, err := ParseCron("@daily")
	if err != nil {
		t.Fatalf("ParseCron shortcut returned error: %v", err)
	}
	next := daily.NextAfter(time.Date(2026, 5, 21, 23, 59, 0, 0, time.UTC))
	want := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("NextAfter(@daily) = %s, want %s", next, want)
	}
}

func TestCronNextAfterBoundaries(t *testing.T) {
	yearly, err := ParseCron("@yearly")
	if err != nil {
		t.Fatalf("ParseCron returned error: %v", err)
	}
	next := yearly.NextAfter(time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC))
	want := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("yearly boundary = %s, want %s", next, want)
	}

	monthly, err := ParseCron("@monthly")
	if err != nil {
		t.Fatalf("ParseCron returned error: %v", err)
	}
	next = monthly.NextAfter(time.Date(2026, 1, 31, 23, 59, 0, 0, time.UTC))
	want = time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("monthly boundary = %s, want %s", next, want)
	}
}

func TestCronInvalidExpressions(t *testing.T) {
	cases := []string{
		"",
		"* * * *",
		"60 * * * *",
		"*/0 * * * *",
		"10-5 * * * *",
		"@reboot",
	}
	for _, tc := range cases {
		if _, err := ParseCron(tc); err == nil {
			t.Fatalf("ParseCron(%q) returned nil error", tc)
		}
	}
}
