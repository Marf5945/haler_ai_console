package hookgene

import "testing"

func TestGeometricEightBuckets(t *testing.T) {
	buckets := make([]float64, PressureBucketCount)
	if PressureBucketCount != 8 {
		t.Fatalf("bucket count = %d, want 8", PressureBucketCount)
	}
	// 全 0 → 幾何平均約等於 epsilon。
	g := GeometricPressure(buckets)
	if g <= 0 || g > 0.01 {
		t.Fatalf("all-zero geometric = %v, want ~epsilon", g)
	}
}

func TestAdaptiveRange(t *testing.T) {
	buckets := []float64{1, 0, 0, 0, 0, 0, 0, 0}
	a := AdaptivePressure(buckets)
	if a < 0 || a > 1 {
		t.Fatalf("adaptive out of range: %v", a)
	}
	// 單桶爆量：dominant=1 → adaptive 約落在 0.4 附近（幾何平均壓抑），且 <0.5。
	if a >= 0.5 {
		t.Fatalf("single-spike adaptive should be suppressed (<0.5), got %v", a)
	}
}
