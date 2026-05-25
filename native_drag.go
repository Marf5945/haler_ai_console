package main

const (
	nativeDragStatusSuccess   = "success"
	nativeDragStatusCancelled = "cancelled"
	nativeDragStatusFailed    = "failed"
)

type nativeDragResult struct {
	Status           string
	FallbackRequired bool
	Message          string
	LandedPath       string
}
