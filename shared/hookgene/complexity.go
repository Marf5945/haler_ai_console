package hookgene

import "math"

// 第 8 桶整合（§3.1.5.18.5）。本檔提供純函式，供 EAS pressure_states 投影使用。
// 維護重點：加入 hook_complexity 後，EAS 幾何平均指數由 1/7 改為 1/8（PressureBucketCount）。
const PressureBucketCount = 8

// epsilon 為幾何平均平滑項，與 EAS §3.1.5.7 一致。
const epsilon = 0.001

// GeometricPressure 計算各桶幾何平均（含 epsilon 平滑）。
// 傳入 8 桶（含 hook_complexity）即為 ^(1/8)。
func GeometricPressure(buckets []float64) float64 {
	if len(buckets) == 0 {
		return 0
	}
	prod := 1.0
	for _, b := range buckets {
		prod *= (b + epsilon)
	}
	return math.Pow(prod, 1.0/float64(len(buckets)))
}

// AdaptivePressure = geometric*0.6 + dominant*0.4（權重和=1，範圍維持 [0,1]）。
func AdaptivePressure(buckets []float64) float64 {
	g := GeometricPressure(buckets)
	dominant := 0.0
	for _, b := range buckets {
		if b > dominant {
			dominant = b
		}
	}
	return g*0.6 + dominant*0.4
}
