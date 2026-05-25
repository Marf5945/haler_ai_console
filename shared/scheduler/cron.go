// Package scheduler 提供 cron 排程表達式的解析與時間匹配功能。
// 設計原則：
//   - 零外部依賴：僅使用 Go 標準函式庫（strings, strconv, fmt, time）
//   - 標準 cron 格式：支援五欄位格式（分 時 日 月 週）
//   - 安全性：NextAfter 設有四年上限，避免無窮迴圈
package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// --------------------------------------------------------------------------
// 各欄位的合法範圍定義
// --------------------------------------------------------------------------

// fieldRange 定義單一 cron 欄位的最小值與最大值。
type fieldRange struct {
	min int
	max int
}

// cronFieldRanges 依序為：分鐘、小時、日、月、星期。
var cronFieldRanges = [5]fieldRange{
	{0, 59},  // 分鐘
	{0, 23},  // 小時
	{1, 31},  // 日
	{1, 12},  // 月
	{0, 6},   // 星期（0=週日, 6=週六）
}

// cronFieldNames 用於錯誤訊息，與 cronFieldRanges 同序。
var cronFieldNames = [5]string{
	"minute（分鐘）",
	"hour（小時）",
	"day（日）",
	"month（月）",
	"weekday（星期）",
}

// --------------------------------------------------------------------------
// 快捷別名
// --------------------------------------------------------------------------

// shortcuts 將 cron 快捷別名對應到標準五欄位表達式。
var shortcuts = map[string]string{
	"@yearly":  "0 0 1 1 *",
	"@monthly": "0 0 1 * *",
	"@weekly":  "0 0 * * 0",
	"@daily":   "0 0 * * *",
	"@hourly":  "0 * * * *",
}

// --------------------------------------------------------------------------
// CronExpr — 解析後的 cron 表達式
// --------------------------------------------------------------------------

// CronExpr 儲存解析完成的 cron 表達式。
// 每個欄位為一組已排序的允許值（[]int），供匹配與推算使用。
type CronExpr struct {
	Minute  []int // 允許的分鐘值（0–59）
	Hour    []int // 允許的小時值（0–23）
	Day     []int // 允許的日值（1–31）
	Month   []int // 允許的月份值（1–12）
	Weekday []int // 允許的星期值（0–6，0=週日）
}

// --------------------------------------------------------------------------
// ParseCron — 解析 cron 表達式字串
// --------------------------------------------------------------------------

// ParseCron 將五欄位 cron 表達式（或快捷別名）解析為 *CronExpr。
//
// 支援的語法：
//   - "*"       任意值
//   - "*/N"     每隔 N（步進）
//   - "1-5"     範圍（含頭尾）
//   - "1,3,5"   列舉
//   - "1-5,10,*/15"  以上組合（以逗號分隔）
//
// 支援的快捷別名：@yearly、@monthly、@weekly、@daily、@hourly。
//
// 回傳錯誤情境：欄位數量不足、數值超出合法範圍、格式不合法。
func ParseCron(expr string) (*CronExpr, error) {
	expr = strings.TrimSpace(expr)

	// 處理快捷別名
	if strings.HasPrefix(expr, "@") {
		expanded, ok := shortcuts[strings.ToLower(expr)]
		if !ok {
			return nil, fmt.Errorf("scheduler: 不支援的快捷別名 %q", expr)
		}
		expr = expanded
	}

	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("scheduler: cron 表達式須恰好五個欄位，收到 %d 個", len(parts))
	}

	// 依序解析五個欄位
	parsed := make([][]int, 5)
	for i, part := range parts {
		vals, err := parseField(part, cronFieldRanges[i])
		if err != nil {
			return nil, fmt.Errorf("scheduler: 解析 %s 欄位 %q 失敗: %w", cronFieldNames[i], part, err)
		}
		parsed[i] = vals
	}

	return &CronExpr{
		Minute:  parsed[0],
		Hour:    parsed[1],
		Day:     parsed[2],
		Month:   parsed[3],
		Weekday: parsed[4],
	}, nil
}

// --------------------------------------------------------------------------
// parseField — 解析單一欄位（支援逗號分隔的多組子表達式）
// --------------------------------------------------------------------------

// parseField 將單一欄位字串解析為一組排序且去重的整數值。
// 子表達式以逗號分隔，每組可為 *、*/N、A-B、A-B/N 或純數字。
func parseField(field string, fr fieldRange) ([]int, error) {
	// 使用 map 去重
	set := make(map[int]bool)

	segments := strings.Split(field, ",")
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return nil, fmt.Errorf("空的子表達式")
		}

		vals, err := parseSegment(seg, fr)
		if err != nil {
			return nil, err
		}
		for _, v := range vals {
			set[v] = true
		}
	}

	// 轉為排序後的切片
	result := sortedKeys(set, fr)
	return result, nil
}

// --------------------------------------------------------------------------
// parseSegment — 解析單一子表達式（不含逗號）
// --------------------------------------------------------------------------

// parseSegment 處理一組不含逗號的子表達式，回傳其展開後的整數值。
//
// 支援格式：
//   - "*"       展開為 min..max
//   - "*/N"     從 min 開始每隔 N
//   - "A-B"     從 A 到 B（含）
//   - "A-B/N"   從 A 到 B 每隔 N
//   - "N"       單一數值
func parseSegment(seg string, fr fieldRange) ([]int, error) {
	// 檢查是否有步進 "/"
	var stepStr string
	base := seg
	if idx := strings.Index(seg, "/"); idx != -1 {
		base = seg[:idx]
		stepStr = seg[idx+1:]
	}

	// 決定範圍起止
	var rangeMin, rangeMax int

	switch {
	case base == "*":
		// 萬用字元：整個合法範圍
		rangeMin = fr.min
		rangeMax = fr.max

	case strings.Contains(base, "-"):
		// 範圍表達式 "A-B"
		dashParts := strings.SplitN(base, "-", 2)
		var err error
		rangeMin, err = strconv.Atoi(dashParts[0])
		if err != nil {
			return nil, fmt.Errorf("無法解析範圍起始值 %q: %w", dashParts[0], err)
		}
		rangeMax, err = strconv.Atoi(dashParts[1])
		if err != nil {
			return nil, fmt.Errorf("無法解析範圍結束值 %q: %w", dashParts[1], err)
		}
		if rangeMin > rangeMax {
			return nil, fmt.Errorf("範圍起始 %d 大於結束 %d", rangeMin, rangeMax)
		}

	default:
		// 單一數值（可能帶步進，但無範圍則視為單值）
		if stepStr != "" {
			return nil, fmt.Errorf("步進語法 %q 須搭配 * 或範圍使用", seg)
		}
		val, err := strconv.Atoi(base)
		if err != nil {
			return nil, fmt.Errorf("無法解析數值 %q: %w", base, err)
		}
		if val < fr.min || val > fr.max {
			return nil, fmt.Errorf("數值 %d 超出合法範圍 %d–%d", val, fr.min, fr.max)
		}
		return []int{val}, nil
	}

	// 驗證範圍邊界
	if rangeMin < fr.min || rangeMax > fr.max {
		return nil, fmt.Errorf("範圍 %d–%d 超出合法範圍 %d–%d", rangeMin, rangeMax, fr.min, fr.max)
	}

	// 計算步進值（預設為 1）
	step := 1
	if stepStr != "" {
		var err error
		step, err = strconv.Atoi(stepStr)
		if err != nil {
			return nil, fmt.Errorf("無法解析步進值 %q: %w", stepStr, err)
		}
		if step <= 0 {
			return nil, fmt.Errorf("步進值須為正整數，收到 %d", step)
		}
	}

	// 展開範圍
	var vals []int
	for v := rangeMin; v <= rangeMax; v += step {
		vals = append(vals, v)
	}
	return vals, nil
}

// --------------------------------------------------------------------------
// Matches — 檢查指定時間是否匹配
// --------------------------------------------------------------------------

// Matches 判斷時間 t 是否符合此 cron 表達式。
// 僅比對分鐘、小時、日、月、星期，忽略秒數。
func (c *CronExpr) Matches(t time.Time) bool {
	return contains(c.Minute, t.Minute()) &&
		contains(c.Hour, t.Hour()) &&
		contains(c.Day, t.Day()) &&
		contains(c.Month, int(t.Month())) &&
		contains(c.Weekday, int(t.Weekday()))
}

// --------------------------------------------------------------------------
// NextAfter — 計算下一個觸發時間
// --------------------------------------------------------------------------

// NextAfter 回傳在時間 t 之後、最近的下一個符合此 cron 表達式的時間點。
//
// 演算法：
//  1. 從 t 的下一分鐘開始（秒歸零）
//  2. 依序嘗試年→月→日→時→分，若當前值不在允許集合中則進位
//  3. 安全機制：若四年內未找到匹配，回傳零值 time.Time（避免無窮迴圈）
//
// 回傳零值 time.Time 表示在合理範圍內找不到下一次觸發時間。
func (c *CronExpr) NextAfter(t time.Time) time.Time {
	// 從下一分鐘開始，秒與奈秒歸零
	candidate := t.Truncate(time.Minute).Add(time.Minute)

	// 安全上限：四年後
	limit := t.AddDate(4, 0, 0)

	for candidate.Before(limit) {
		// --- 月份檢查 ---
		if !contains(c.Month, int(candidate.Month())) {
			// 跳至下一個允許的月份的第一天 00:00
			candidate = c.advanceMonth(candidate)
			continue
		}

		// --- 日與星期檢查 ---
		if !contains(c.Day, candidate.Day()) || !contains(c.Weekday, int(candidate.Weekday())) {
			// 跳至下一天 00:00
			candidate = advanceDay(candidate)
			continue
		}

		// --- 小時檢查 ---
		if !contains(c.Hour, candidate.Hour()) {
			// 跳至下一個允許的小時 :00
			candidate = c.advanceHour(candidate)
			continue
		}

		// --- 分鐘檢查 ---
		if !contains(c.Minute, candidate.Minute()) {
			// 跳至下一個允許的分鐘
			candidate = c.advanceMinute(candidate)
			continue
		}

		// 所有欄位皆匹配
		return candidate
	}

	// 四年內未找到匹配，回傳零值
	return time.Time{}
}

// --------------------------------------------------------------------------
// 內部進位輔助函式
// --------------------------------------------------------------------------

// advanceMonth 從 candidate 跳至下一個允許月份的第一天 00:00。
// 若當年已無允許月份，則進入下一年。
func (c *CronExpr) advanceMonth(candidate time.Time) time.Time {
	year := candidate.Year()
	month := int(candidate.Month())

	// 嘗試在當年找下一個允許的月份
	next := nextInSet(c.Month, month)
	if next != -1 {
		return time.Date(year, time.Month(next), 1, 0, 0, 0, 0, candidate.Location())
	}

	// 當年無更大的允許月份，跳至下一年最小允許月份
	return time.Date(year+1, time.Month(c.Month[0]), 1, 0, 0, 0, 0, candidate.Location())
}

// advanceDay 將 candidate 推進至下一天 00:00:00。
func advanceDay(candidate time.Time) time.Time {
	next := candidate.AddDate(0, 0, 1)
	return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, candidate.Location())
}

// advanceHour 從 candidate 跳至下一個允許小時的 :00。
// 若當天已無允許小時，則進入下一天 00:00。
func (c *CronExpr) advanceHour(candidate time.Time) time.Time {
	next := nextInSet(c.Hour, candidate.Hour())
	if next != -1 {
		return time.Date(candidate.Year(), candidate.Month(), candidate.Day(),
			next, 0, 0, 0, candidate.Location())
	}
	// 當天已無允許小時，跳至下一天
	return advanceDay(candidate)
}

// advanceMinute 從 candidate 跳至下一個允許分鐘。
// 若當小時已無允許分鐘，則推進至下一分鐘整點（交由外層迴圈重新檢查小時）。
func (c *CronExpr) advanceMinute(candidate time.Time) time.Time {
	next := nextInSet(c.Minute, candidate.Minute())
	if next != -1 {
		return time.Date(candidate.Year(), candidate.Month(), candidate.Day(),
			candidate.Hour(), next, 0, 0, candidate.Location())
	}
	// 當小時已無允許分鐘，跳至下一小時 :00
	return time.Date(candidate.Year(), candidate.Month(), candidate.Day(),
		candidate.Hour()+1, 0, 0, 0, candidate.Location())
}

// --------------------------------------------------------------------------
// 通用工具函式
// --------------------------------------------------------------------------

// contains 檢查排序後的切片 vals 中是否包含 target。
// 使用線性搜尋（欄位值域很小，最多 60 個元素，線性即足夠）。
func contains(vals []int, target int) bool {
	for _, v := range vals {
		if v == target {
			return true
		}
	}
	return false
}

// nextInSet 在已排序的 vals 中找出嚴格大於 current 的最小值。
// 若無符合者回傳 -1。
func nextInSet(vals []int, current int) int {
	for _, v := range vals {
		if v > current {
			return v
		}
	}
	return -1
}

// sortedKeys 將 map 的鍵按升序排列後回傳。
// 利用欄位值域有限（最多 0–59）的特性，以計數排序實現。
func sortedKeys(set map[int]bool, fr fieldRange) []int {
	result := make([]int, 0, len(set))
	for v := fr.min; v <= fr.max; v++ {
		if set[v] {
			result = append(result, v)
		}
	}
	return result
}
