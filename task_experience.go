// task_experience.go — 跨任務經驗回饋（v3.1.8）。
// 任務終局（completed/failed）時寫一筆精簡經驗到 dag_runs/experience.jsonl；
// 下次規劃時把「與新目標相似的過往經驗」做成短摘要注入 planner prompt。
// 記錄恆開（純落盤、不改行為）；注入由 AI_CONSOLE_TASK_EXPERIENCE 控制，預設關。
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ui_console/data/storage"
	"ui_console/orchestration/dag"
)

const (
	taskExperienceFile     = "experience.jsonl"
	taskExperienceMaxKeep  = 100      // 讀取端只看最近 N 筆
	taskExperienceMaxFile  = 256 << 10 // 檔案超過此大小時重寫保留尾段
	taskExperienceMaxMatch = 3        // 注入最多幾筆
)

// TaskExperience 一筆任務經驗（刻意精簡：planner 只需要「什麼任務、成敗、敗在哪」）。
type TaskExperience struct {
	RunID           string `json:"run_id"`
	Title           string `json:"title"`
	Status          string `json:"status"` // completed / failed
	FailedNodeTitle string `json:"failed_node_title,omitempty"`
	FailureCategory string `json:"failure_category,omitempty"`
	NodeCount       int    `json:"node_count"`
	CreatedAt       string `json:"created_at"`
}

// taskExperienceInjectEnabled：注入 flag，預設關（記錄不受此 flag 影響）。
func taskExperienceInjectEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AI_CONSOLE_TASK_EXPERIENCE"))) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// recordTaskExperience 從終局 run 萃取經驗並落盤；失敗只記 trace 不擋主流程。
func recordTaskExperience(projectRoot string, run *dag.DAGRun) {
	if run == nil || (run.Status != "completed" && run.Status != "failed") {
		return
	}
	exp := TaskExperience{
		RunID:     run.ID,
		Title:     truncateRunes(strings.TrimSpace(run.Title), 200),
		Status:    run.Status,
		NodeCount: len(run.Nodes),
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	for _, n := range run.Nodes {
		if n.Status == dag.StatusFailed {
			exp.FailedNodeTitle = truncateRunes(firstNonEmpty(n.Title, n.Target), 120)
			exp.FailureCategory = n.FailureCategory
			break
		}
	}
	_ = appendTaskExperience(projectRoot, exp)
}

func appendTaskExperience(projectRoot string, exp TaskExperience) error {
	dir := filepath.Join(projectRoot, "dag_runs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, taskExperienceFile)
	// 檔案過大 → 重寫只留最近 N 筆（jsonl 無索引，靠定期收斂控制大小）。
	if info, err := os.Stat(path); err == nil && info.Size() > taskExperienceMaxFile {
		if recent := loadRecentTaskExperiences(projectRoot, taskExperienceMaxKeep); len(recent) > 0 {
			var b strings.Builder
			for _, e := range recent {
				line, _ := json.Marshal(e)
				b.Write(line)
				b.WriteString("\n")
			}
			_ = os.WriteFile(path, []byte(b.String()), 0o600)
		}
	}
	line, err := json.Marshal(exp)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

// loadRecentTaskExperiences 讀最近 limit 筆（壞行跳過）。
func loadRecentTaskExperiences(projectRoot string, limit int) []TaskExperience {
	f, err := os.Open(filepath.Join(projectRoot, "dag_runs", taskExperienceFile))
	if err != nil {
		return nil
	}
	defer f.Close()
	var all []TaskExperience
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64<<10), 64<<10)
	for scanner.Scan() {
		var exp TaskExperience
		if json.Unmarshal(scanner.Bytes(), &exp) == nil && exp.Title != "" {
			all = append(all, exp)
		}
	}
	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all
}

// matchTaskExperiences 用字元 2-gram 重疊找相似任務（中文友善、零依賴）。
// 失敗經驗加權（規劃時避雷比複製成功更有價值），新經驗優先。
func matchTaskExperiences(goal string, exps []TaskExperience, max int) []TaskExperience {
	goalGrams := charBigrams(goal)
	if len(goalGrams) == 0 {
		return nil
	}
	type scored struct {
		exp   TaskExperience
		score int
	}
	var candidates []scored
	for _, exp := range exps {
		overlap := 0
		for gram := range charBigrams(exp.Title) {
			if goalGrams[gram] {
				overlap++
			}
		}
		if overlap < 2 { // 至少兩個 2-gram 重疊才算相似
			continue
		}
		if exp.Status == "failed" {
			overlap += 2
		}
		candidates = append(candidates, scored{exp, overlap})
	}
	// 穩定排序：分數高優先，同分新的優先（candidates 本身按時間舊→新）
	var out []TaskExperience
	for len(out) < max && len(candidates) > 0 {
		best := 0
		for i := range candidates {
			if candidates[i].score >= candidates[best].score {
				best = i
			}
		}
		out = append(out, candidates[best].exp)
		candidates = append(candidates[:best], candidates[best+1:]...)
	}
	return out
}

func charBigrams(s string) map[string]bool {
	runes := []rune(strings.ToLower(strings.TrimSpace(s)))
	grams := map[string]bool{}
	for i := 0; i+1 < len(runes); i++ {
		grams[string(runes[i:i+2])] = true
	}
	return grams
}

// taskExperienceDigest 給 planner 的注入段；flag 關或無相似經驗回空字串。
func (a *App) taskExperienceDigest(goal string) string {
	if !taskExperienceInjectEnabled() {
		return ""
	}
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	matches := matchTaskExperiences(goal, loadRecentTaskExperiences(projectRoot, taskExperienceMaxKeep), taskExperienceMaxMatch)
	if len(matches) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n過去類似任務的經驗（規劃時參考，不要照抄）：\n")
	for _, exp := range matches {
		if exp.Status == "failed" {
			fmt.Fprintf(&b, "- 「%s」失敗於「%s」（%s），規劃時避免同樣安排。\n", exp.Title, exp.FailedNodeTitle, exp.FailureCategory)
		} else {
			fmt.Fprintf(&b, "- 「%s」曾成功完成（%d 步）。\n", exp.Title, exp.NodeCount)
		}
	}
	return b.String()
}
