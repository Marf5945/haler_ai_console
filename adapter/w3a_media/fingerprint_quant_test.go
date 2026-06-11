package w3a_media

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestReadWAVSamplesKeepsInt16PCM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.wav")
	want := []int16{0, 32767, -32768, 1}
	if err := writeTestWAV(path, want); err != nil {
		t.Fatalf("write wav: %v", err)
	}

	got, sampleRate, err := readWAVSamples(path)
	if err != nil {
		t.Fatalf("readWAVSamples: %v", err)
	}
	if sampleRate != 8000 {
		t.Fatalf("sampleRate = %d, want 8000", sampleRate)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sample[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestAudioQuantizedSampleScoring(t *testing.T) {
	if e := sampleEnergy(0); e != 0 {
		t.Fatalf("zero energy = %v, want 0", e)
	}
	if e := sampleEnergy(32767); e < 0.99 || e > 1.01 {
		t.Fatalf("full scale energy = %v, want ~1", e)
	}
	if score := detectAudioLSB([]int16{1, 3, 2, 4}); score > 0.001 {
		t.Fatalf("balanced LSB score = %v, want ~0", score)
	}
}

func writeTestWAV(path string, samples []int16) error {
	dataSize := uint32(len(samples) * 2)
	header := make([]byte, 44)
	copy(header[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(header[4:8], 36+dataSize)
	copy(header[8:12], []byte("WAVE"))
	copy(header[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	binary.LittleEndian.PutUint16(header[22:24], 1)
	binary.LittleEndian.PutUint32(header[24:28], 8000)
	binary.LittleEndian.PutUint32(header[28:32], 8000*2)
	binary.LittleEndian.PutUint16(header[32:34], 2)
	binary.LittleEndian.PutUint16(header[34:36], 16)
	copy(header[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(header[40:44], dataSize)

	data := make([]byte, 44+dataSize)
	copy(data, header)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(data[44+i*2:46+i*2], uint16(sample))
	}
	return os.WriteFile(path, data, 0o600)
}
