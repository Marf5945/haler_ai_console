package voice

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ui_console/shared/executil"
)

const maxAudioBytes = 16 * 1024 * 1024
const managedModelURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin"
const managedModelSHA256 = "60ed5bc3dd14eea856493d334349b405782ddcaf0028d4b5df4088345fba2efe"

type TranscriptResult struct {
	Text            string  `json:"text"`
	Language        string  `json:"language"`
	DebugSaved      bool    `json:"debugSaved"`
	Warning         string  `json:"warning,omitempty"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	AudioPath       string  `json:"audioPath,omitempty"`
}

func (s *Service) TranscribeWAVBase64(ctx context.Context, audioBase64, mimeType, panelLanguage string) (TranscriptResult, error) {
	state := s.Get(panelLanguage)
	if !state.WhisperAvailable || !state.ModelAvailable {
		return TranscriptResult{}, fmt.Errorf("voice: whisper not ready: %s", state.Status)
	}
	if strings.TrimSpace(audioBase64) == "" {
		return TranscriptResult{}, fmt.Errorf("voice: empty audio payload")
	}
	// SEC-21: 先用 base64 長度估算，拒絕超限 payload（避免先分配大記憶體）
	maxBase64Len := maxAudioBytes*4/3 + 4
	if len(audioBase64) > maxBase64Len {
		return TranscriptResult{}, fmt.Errorf("voice: audio payload too large (base64 len=%d)", len(audioBase64))
	}
	audioBytes, err := base64.StdEncoding.DecodeString(audioBase64)
	if err != nil {
		return TranscriptResult{}, fmt.Errorf("voice: decode wav payload: %w", err)
	}
	if mimeType != "" && !strings.Contains(strings.ToLower(mimeType), "wav") {
		return TranscriptResult{}, fmt.Errorf("voice: unsupported audio type %q", mimeType)
	}

	// WAV 標頭驗證 + 時長檢查
	sampleRate, durationSec, wavWarning, headerErr := validateWAVHeader(audioBytes)
	if headerErr != nil {
		return TranscriptResult{}, headerErr
	}
	_ = sampleRate // 過渡期記錄但不擋非 16kHz

	debugMode := state.Settings.DebugMode
	workDir := filepath.Join(s.root, "data", "voice", "tmp")
	if debugMode {
		workDir = filepath.Join(s.root, "data", "voice", "debug")
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return TranscriptResult{}, fmt.Errorf("voice: create work dir: %w", err)
	}

	// SEC-22: 使用 os.CreateTemp 避免可預測檔名
	wavFile, err := os.CreateTemp(workDir, "voice-*.wav")
	if err != nil {
		return TranscriptResult{}, fmt.Errorf("voice: create temp wav: %w", err)
	}
	wavPath := wavFile.Name()
	outPrefix := strings.TrimSuffix(wavPath, ".wav")
	if _, err := wavFile.Write(audioBytes); err != nil {
		wavFile.Close()
		return TranscriptResult{}, fmt.Errorf("voice: write wav: %w", err)
	}
	wavFile.Close()
	if !debugMode {
		defer os.Remove(wavPath)
		defer os.Remove(outPrefix + ".txt")
	}

	// 動態 timeout：≤30s 音訊用 60s，>30s 用 audioSeconds*3（上限 600s）
	var transcribeTimeout time.Duration
	if durationSec <= 30 {
		transcribeTimeout = 60 * time.Second
	} else {
		transcribeTimeout = time.Duration(durationSec*3) * time.Second
		if transcribeTimeout > 600*time.Second {
			transcribeTimeout = 600 * time.Second
		}
	}
	transcribeCtx, transcribeCancel := context.WithTimeout(ctx, transcribeTimeout)
	defer transcribeCancel()

	text, err := runWhisper(transcribeCtx, state.WhisperBinPath, state.ModelPath, wavPath, outPrefix, state.Language)
	if err != nil {
		return TranscriptResult{}, err
	}
	result := TranscriptResult{
		Text:            strings.TrimSpace(text),
		Language:        state.Language,
		Warning:         wavWarning,
		DurationSeconds: durationSec,
		DebugSaved:      debugMode,
	}
	if debugMode {
		result.AudioPath = wavPath
		_ = writeDebugTranscript(outPrefix+".json", result)
		_ = s.pruneDebugFiles(5)
	}
	return result, nil
}

func (s *Service) InstallBaseModel(ctx context.Context, panelLanguage string) (State, error) {
	if err := validateManagedModelURL(managedModelURL); err != nil {
		return s.Get(panelLanguage), err
	}
	target := s.ManagedModelPath()
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return s.Get(panelLanguage), fmt.Errorf("voice: create model dir: %w", err)
	}
	tmp := target + ".download"
	_ = os.Remove(tmp)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, managedModelURL, nil)
	if err != nil {
		return s.Get(panelLanguage), fmt.Errorf("voice: create model download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return s.Get(panelLanguage), fmt.Errorf("voice: download base model: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return s.Get(panelLanguage), fmt.Errorf("voice: download base model: HTTP %d", resp.StatusCode)
	}

	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return s.Get(panelLanguage), fmt.Errorf("voice: create model file: %w", err)
	}
	hasher := sha256.New()
	if _, err = io.Copy(io.MultiWriter(file, hasher), resp.Body); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return s.Get(panelLanguage), fmt.Errorf("voice: write model file: %w", err)
	}
	if sum := hex.EncodeToString(hasher.Sum(nil)); sum != managedModelSHA256 {
		_ = file.Close()
		_ = os.Remove(tmp)
		return s.Get(panelLanguage), fmt.Errorf("voice: model checksum mismatch; expected %s, got %s", managedModelSHA256, sum)
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(tmp)
		return s.Get(panelLanguage), fmt.Errorf("voice: close model file: %w", err)
	}
	if err = os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return s.Get(panelLanguage), fmt.Errorf("voice: install model file: %w", err)
	}
	return s.Get(panelLanguage), nil
}

func (s *Service) RemoveManagedModel(panelLanguage string) (State, error) {
	target := s.ManagedModelPath()
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return s.Get(panelLanguage), fmt.Errorf("voice: remove model: %w", err)
	}
	_ = os.Remove(target + ".download")
	return s.Get(panelLanguage), nil
}

func validateManagedModelURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("voice: invalid model URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "huggingface.co" || parsed.Path != "/ggerganov/whisper.cpp/resolve/main/ggml-base.bin" {
		return fmt.Errorf("voice: model URL is not in the built-in allowlist")
	}
	return nil
}

func verifyManagedModelFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}
	if sum := hex.EncodeToString(hasher.Sum(nil)); sum != managedModelSHA256 {
		return fmt.Errorf("voice: model checksum mismatch")
	}
	return nil
}

func runWhisper(ctx context.Context, binPath, modelPath, wavPath, outPrefix, lang string) (string, error) {
	args := []string{"-m", modelPath, "-f", wavPath, "-otxt", "-of", outPrefix, "-nt"}
	if lang != "" {
		args = append(args, "-l", lang)
	}
	cmd := executil.CommandContext(ctx, binPath, args...)
	output, err := cmd.CombinedOutput()
	txtPath := outPrefix + ".txt"
	if data, readErr := os.ReadFile(txtPath); readErr == nil {
		text := strings.TrimSpace(string(data))
		if text != "" {
			return text, nil
		}
	}
	if err != nil {
		return "", fmt.Errorf("voice: whisper failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return "", fmt.Errorf("voice: no speech recognized")
}

func writeDebugTranscript(path string, result TranscriptResult) error {
	payload, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

func (s *Service) ClearDebug(panelLanguage string) (State, error) {
	debugDir := filepath.Join(s.root, "data", "voice", "debug")
	if err := os.RemoveAll(debugDir); err != nil {
		return s.Get(panelLanguage), fmt.Errorf("voice: clear debug files: %w", err)
	}
	return s.Get(panelLanguage), nil
}

func (s *Service) pruneDebugFiles(limit int) error {
	debugDir := filepath.Join(s.root, "data", "voice", "debug")
	entries, err := os.ReadDir(debugDir)
	if err != nil {
		return nil
	}
	type group struct {
		prefix string
		mod    time.Time
	}
	groups := map[string]group{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "voice-") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".wav"), ".json")
		name = strings.TrimSuffix(name, ".txt")
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if current, ok := groups[name]; !ok || info.ModTime().After(current.mod) {
			groups[name] = group{prefix: name, mod: info.ModTime()}
		}
	}
	list := make([]group, 0, len(groups))
	for _, item := range groups {
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].mod.After(list[j].mod)
	})
	for _, item := range list[limit:] {
		_ = os.Remove(filepath.Join(debugDir, item.prefix+".wav"))
		_ = os.Remove(filepath.Join(debugDir, item.prefix+".txt"))
		_ = os.Remove(filepath.Join(debugDir, item.prefix+".json"))
	}
	return nil
}

// validateWAVHeader 驗證 WAV 檔案標頭，回傳取樣率、時長、警告訊息。
// 過渡期接受 16000/44100/48000 Hz，非 16000 會回傳 warning。
func validateWAVHeader(data []byte) (sampleRate uint32, durationSec float64, warning string, err error) {
	if len(data) < 44 {
		return 0, 0, "", fmt.Errorf("voice: WAV 標頭太短 (%d bytes)", len(data))
	}
	// RIFF magic
	if string(data[0:4]) != "RIFF" {
		return 0, 0, "", fmt.Errorf("voice: 缺少 RIFF 標頭")
	}
	// WAVE format
	if string(data[8:12]) != "WAVE" {
		return 0, 0, "", fmt.Errorf("voice: 缺少 WAVE 格式標記")
	}
	// fmt chunk — PCM = 1
	format := binary.LittleEndian.Uint16(data[20:22])
	if format != 1 {
		return 0, 0, "", fmt.Errorf("voice: 不支援的音訊格式 %d，僅接受 PCM (1)", format)
	}
	// Mono only
	channels := binary.LittleEndian.Uint16(data[22:24])
	if channels != 1 {
		return 0, 0, "", fmt.Errorf("voice: 不支援 %d 聲道，僅接受單聲道 (mono)", channels)
	}
	// Sample rate — transitional: accept 16000/44100/48000
	sr := binary.LittleEndian.Uint32(data[24:28])
	switch sr {
	case 16000:
		// ideal, no warning
	case 44100, 48000:
		warning = fmt.Sprintf("non-16kHz input (%d Hz), consider upgrading client", sr)
	default:
		return 0, 0, "", fmt.Errorf("voice: 不支援的取樣率 %d Hz，僅接受 16000/44100/48000", sr)
	}
	// Bits per sample
	bps := binary.LittleEndian.Uint16(data[34:36])
	if bps != 16 {
		return 0, 0, "", fmt.Errorf("voice: 不支援 %d-bit，僅接受 16-bit PCM", bps)
	}
	// 掃描 data chunk 計算時長
	dataSize := uint32(0)
	for i := 36; i+8 <= len(data); i++ {
		if string(data[i:i+4]) == "data" {
			dataSize = binary.LittleEndian.Uint32(data[i+4 : i+8])
			break
		}
	}
	if dataSize == 0 && len(data) > 44 {
		// fallback: assume data starts at 44
		dataSize = uint32(len(data) - 44)
	}
	bytesPerSample := uint32(channels) * uint32(bps) / 8
	if bytesPerSample > 0 && sr > 0 {
		totalSamples := dataSize / bytesPerSample
		durationSec = float64(totalSamples) / float64(sr)
	}
	// 120 秒上限
	if durationSec > 120.0 {
		return sr, durationSec, warning, fmt.Errorf("voice: 音訊時長 %.1f 秒超過上限 120 秒", durationSec)
	}
	return sr, durationSec, warning, nil
}
