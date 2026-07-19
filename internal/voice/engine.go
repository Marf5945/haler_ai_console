package voice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// VoiceEngine and Audio Queue (spec §34.5 / §34.6).
//
// Queue semantics: each Voice Job snapshots its voice_id at creation time.
// Switching persona affects only jobs created afterwards; queued or playing
// jobs keep their original voice profile. Voice output never changes
// permissions, risk policy, tool availability, or task authority.

type VoiceJobKind string

const (
	JobKindProbe        VoiceJobKind = "probe"
	JobKindReadout      VoiceJobKind = "readout"
	JobKindSafetyNotice VoiceJobKind = "safety_notice"
)

type VoiceJobStatus string

const (
	JobQueued    VoiceJobStatus = "queued"
	JobPlaying   VoiceJobStatus = "playing"
	JobDone      VoiceJobStatus = "done"
	JobCancelled VoiceJobStatus = "cancelled"
	JobFailed    VoiceJobStatus = "failed"
	JobSkipped   VoiceJobStatus = "skipped"
)

type VoiceJob struct {
	ID        string         `json:"id"`
	Kind      VoiceJobKind   `json:"kind"`
	Text      string         `json:"text"`
	VoiceID   string         `json:"voiceId"`
	PersonaID string         `json:"personaId,omitempty"`
	Status    VoiceJobStatus `json:"status"`
	Reason    string         `json:"reason,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`

	profile VoiceProfile
}

type SynthRequest struct {
	Text    string
	Profile VoiceProfile
}

// Synthesizer is the platform TTS boundary. Implementations must be
// OS-native (spec §34.6): no third-party TTS engine without a Developer
// Review Card and the §34.8 acquisition flow.
type Synthesizer interface {
	Name() string
	Available() bool
	Speak(ctx context.Context, req SynthRequest) error
}

var (
	ErrTTSUnavailable = errors.New("voice tts: no speech synthesizer available on this platform")
	ErrTTSDisabled    = errors.New("voice tts: voice output is disabled")
	ErrEngineClosed   = errors.New("voice tts: engine closed")
	ErrEmptyText      = errors.New("voice tts: empty text")
)

type EngineStatus struct {
	Enabled     bool      `json:"enabled"`
	Available   bool      `json:"available"`
	Engine      string    `json:"engine"`
	QueueLength int       `json:"queueLength"`
	Current     *VoiceJob `json:"current,omitempty"`
}

const maxVoiceJobTextRunes = 4000

type Engine struct {
	mu            sync.Mutex
	synth         Synthesizer
	enabledFn     func() bool
	queue         []*VoiceJob
	current       *VoiceJob
	cancelCurrent context.CancelFunc
	wake          chan struct{}
	closed        bool
	seq           int64
}

// NewEngine starts the voice output worker. enabledFn gates playback and is
// consulted at enqueue time; pass nil to treat output as always enabled
// (tests only).
func NewEngine(synth Synthesizer, enabledFn func() bool) *Engine {
	e := &Engine{
		synth:     synth,
		enabledFn: enabledFn,
		wake:      make(chan struct{}, 1),
	}
	go e.run()
	return e
}

func (e *Engine) run() {
	for {
		e.mu.Lock()
		if e.closed {
			for _, job := range e.queue {
				job.Status = JobCancelled
				job.Reason = "engine_closed"
			}
			e.queue = nil
			e.mu.Unlock()
			return
		}
		if len(e.queue) == 0 {
			e.mu.Unlock()
			<-e.wake
			continue
		}
		job := e.queue[0]
		e.queue = e.queue[1:]
		ctx, cancel := context.WithCancel(context.Background())
		job.Status = JobPlaying
		e.current = job
		e.cancelCurrent = cancel
		synth := e.synth
		e.mu.Unlock()

		err := synth.Speak(ctx, SynthRequest{Text: job.Text, Profile: job.profile})
		cancel()

		e.mu.Lock()
		switch {
		case ctx.Err() != nil || errors.Is(err, context.Canceled):
			job.Status = JobCancelled
			if job.Reason == "" {
				job.Reason = "cancelled"
			}
		case err != nil:
			job.Status = JobFailed
			job.Reason = err.Error()
		default:
			job.Status = JobDone
		}
		e.current = nil
		e.cancelCurrent = nil
		e.mu.Unlock()
	}
}

// EnqueueFor creates a Voice Job for the given persona using the provided
// voice profile. The voice_id is snapshotted here; later persona switches do
// not affect this job.
func (e *Engine) EnqueueFor(personaID, text string, kind VoiceJobKind, profile VoiceProfile) (VoiceJob, error) {
	text = strings.TrimSpace(text)
	if runes := []rune(text); len(runes) > maxVoiceJobTextRunes {
		text = string(runes[:maxVoiceJobTextRunes])
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.seq++
	job := &VoiceJob{
		ID:        fmt.Sprintf("vjob-%d-%d", time.Now().UnixMilli(), e.seq),
		Kind:      kind,
		Text:      text,
		VoiceID:   profile.VoiceID,
		PersonaID: personaID,
		CreatedAt: time.Now(),
		profile:   profile,
	}
	if e.closed {
		job.Status = JobSkipped
		job.Reason = "engine_closed"
		return *job, ErrEngineClosed
	}
	if text == "" {
		job.Status = JobSkipped
		job.Reason = "empty_text"
		return *job, ErrEmptyText
	}
	if e.enabledFn != nil && !e.enabledFn() {
		job.Status = JobSkipped
		job.Reason = "tts_disabled"
		return *job, ErrTTSDisabled
	}
	if e.synth == nil || !e.synth.Available() {
		job.Status = JobSkipped
		job.Reason = "tts_unavailable"
		return *job, ErrTTSUnavailable
	}
	job.Status = JobQueued
	e.queue = append(e.queue, job)
	e.wakeLocked()
	return *job, nil
}

// EnqueuePreview enqueues a user-initiated voice preview. An explicit
// preview click is treated as consent, so the TTS-enabled toggle is not
// consulted; synthesizer availability is still required.
func (e *Engine) EnqueuePreview(text string, profile VoiceProfile) (VoiceJob, error) {
	text = strings.TrimSpace(text)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.seq++
	job := &VoiceJob{
		ID:        fmt.Sprintf("vjob-%d-%d", time.Now().UnixMilli(), e.seq),
		Kind:      JobKindReadout,
		Text:      text,
		VoiceID:   profile.VoiceID,
		CreatedAt: time.Now(),
		profile:   profile,
	}
	if e.closed {
		job.Status = JobSkipped
		job.Reason = "engine_closed"
		return *job, ErrEngineClosed
	}
	if text == "" {
		job.Status = JobSkipped
		job.Reason = "empty_text"
		return *job, ErrEmptyText
	}
	if e.synth == nil || !e.synth.Available() {
		job.Status = JobSkipped
		job.Reason = "tts_unavailable"
		return *job, ErrTTSUnavailable
	}
	job.Status = JobQueued
	e.queue = append(e.queue, job)
	e.wakeLocked()
	return *job, nil
}

// CancelProbes cancels queued and currently playing probe jobs. Called when
// the user resumes speaking: spoken probes must never overlap active user
// speech (spec §34.6). Readout and safety-notice jobs are not affected.
func (e *Engine) CancelProbes() EngineStatus {
	e.mu.Lock()
	kept := e.queue[:0]
	for _, job := range e.queue {
		if job.Kind == JobKindProbe {
			job.Status = JobCancelled
			job.Reason = "user_resumed_speech"
			continue
		}
		kept = append(kept, job)
	}
	e.queue = kept
	if e.current != nil && e.current.Kind == JobKindProbe && e.cancelCurrent != nil {
		e.current.Reason = "user_resumed_speech"
		e.cancelCurrent()
	}
	status := e.statusLocked()
	e.mu.Unlock()
	return status
}

// StopAll implements the universal stop boundary for voice output: cancel
// the playing job and drop everything queued.
func (e *Engine) StopAll() EngineStatus {
	e.mu.Lock()
	for _, job := range e.queue {
		job.Status = JobCancelled
		job.Reason = "user_stop"
	}
	e.queue = nil
	if e.cancelCurrent != nil {
		if e.current != nil {
			e.current.Reason = "user_stop"
		}
		e.cancelCurrent()
	}
	status := e.statusLocked()
	e.mu.Unlock()
	return status
}

func (e *Engine) Status() EngineStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.statusLocked()
}

func (e *Engine) statusLocked() EngineStatus {
	status := EngineStatus{
		Enabled:     e.enabledFn == nil || e.enabledFn(),
		Available:   e.synth != nil && e.synth.Available(),
		QueueLength: len(e.queue),
	}
	if e.synth != nil {
		status.Engine = e.synth.Name()
	}
	if e.current != nil {
		snapshot := *e.current
		status.Current = &snapshot
	}
	return status
}

// Close shuts the engine down and cancels any active playback.
func (e *Engine) Close() {
	e.mu.Lock()
	e.closed = true
	if e.cancelCurrent != nil {
		e.cancelCurrent()
	}
	e.wakeLocked()
	e.mu.Unlock()
}

func (e *Engine) wakeLocked() {
	select {
	case e.wake <- struct{}{}:
	default:
	}
}
