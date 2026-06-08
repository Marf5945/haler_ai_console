// service.go — 排程引擎核心，負責 Ticker 循環、觸發判定、重疊跳過、失敗重試與補執行。
//
// 設計原則：
//   - 分鐘級精度：使用 time.Ticker（1 分鐘間隔）掃描所有 Job
//   - 重疊保護：同一 Job 正在執行中時跳過新觸發，避免資源爆炸
//   - 容錯機制：失敗自動重試一次（間隔 5 秒），重試也失敗則記錄錯誤
//   - 補執行：啟動時檢查錯過的 Job，只補最近一次，不連鎖補回
//   - 零外部依賴：僅使用 Go 標準函式庫
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ui_console/shared/eventbus"
)

// --------------------------------------------------------------------------
// 外部依賴抽象介面（避免 scheduler 直接依賴 domain 層）
// --------------------------------------------------------------------------

// SchedulerReviewCreator 建立 Review Card 的抽象介面（避免 scheduler 直接依賴 domain/review）。
type SchedulerReviewCreator interface {
	CreateSchedulerReviewCard(jobID, jobName, riskClass, reason string) error
}

// SchedulerAuthChecker 檢查排程授權的抽象介面（避免直接依賴 domain/controlled_trust）。
type SchedulerAuthChecker interface {
	IsAuthorizedForJob(jobID string) bool
}

// --------------------------------------------------------------------------
// 常數定義
// --------------------------------------------------------------------------

const (
	// tickInterval 為排程器掃描間隔。
	tickInterval = 1 * time.Minute

	// retryDelay 為失敗後重試的等待時間。
	retryDelay = 5 * time.Second
)

// --------------------------------------------------------------------------
// ServiceConfig — 排程器初始化設定
// --------------------------------------------------------------------------

// ServiceConfig 包含建立 Service 所需的所有依賴。
type ServiceConfig struct {
	// DataRoot 為資料根目錄，持久化檔案存放於 <DataRoot>/data/scheduler/。
	DataRoot string

	// EventBus 為事件匯流排實例，用於發送排程事件通知。
	EventBus *eventbus.Bus

	// SkillExec 為技能執行器介面，用於 Skill 類型動作。可為 nil（若不需要 Skill 動作）。
	SkillExec SkillExecutor

	// Callbacks 為回呼註冊表。若為 nil，會自動建立空的註冊表。
	Callbacks *CallbackRegistry

	// ReviewCreator 建立 Review Card（可為 nil，無 review 功能時）。
	ReviewCreator SchedulerReviewCreator

	// AuthChecker 檢查排程授權（可為 nil，無授權檢查時）。
	AuthChecker SchedulerAuthChecker

	// ProjectID 目前專案 ID。
	ProjectID string
}

// --------------------------------------------------------------------------
// Service — 排程引擎主結構
// --------------------------------------------------------------------------

// Service 是排程引擎的核心結構，管理所有 Job 的生命週期。
//
// 主要職責：
//   - 啟動 / 停止 Ticker 循環
//   - 每分鐘掃描並觸發到期的 Job
//   - 維護 Job 的 running 狀態（重疊跳過）
//   - 失敗時自動重試一次
//   - 啟動時補執行錯過的 Job
//   - 持久化 Job 狀態與執行歷史
type Service struct {
	// mu 保護 jobs 切片的讀寫。
	mu sync.Mutex

	// jobs 為所有排程任務（包含已停用的）。
	jobs []*Job

	// running 追蹤正在執行中的 Job ID → 取消函式，用於重疊跳過與單任務取消。
	running map[string]context.CancelFunc

	// runMu 保護 running map 的讀寫。
	runMu sync.Mutex

	// store 負責 Job 與執行歷史的 JSON 持久化。
	store *Store

	// resolver 根據 ActionType 回傳對應的 Action 實作。
	resolver *ActionResolver

	// callbacks 為回呼函式註冊表，對外暴露供模組註冊。
	callbacks *CallbackRegistry

	// eventBus 用於發送排程事件通知給前端。
	eventBus *eventbus.Bus

	// reviewCreator 建立 Review Card（風險閘門使用）。
	reviewCreator SchedulerReviewCreator

	// authChecker 檢查排程授權（風險閘門使用）。
	authChecker SchedulerAuthChecker

	// cancel 用於停止 Ticker goroutine。
	cancel context.CancelFunc

	// wg 等待所有 goroutine 結束（Ticker + 正在執行的 Job）。
	wg sync.WaitGroup
}

// --------------------------------------------------------------------------
// NewService — 建立排程引擎
// --------------------------------------------------------------------------

// NewService 建立一個新的排程引擎。
// 此時尚未啟動 Ticker，需呼叫 Start() 開始排程。
func NewService(cfg ServiceConfig) *Service {
	// 確保 callbacks 不為 nil
	cb := cfg.Callbacks
	if cb == nil {
		cb = NewCallbackRegistry()
	}

	return &Service{
		jobs:          make([]*Job, 0),
		running:       make(map[string]context.CancelFunc),
		store:         NewStore(cfg.DataRoot),
		resolver:      NewActionResolver(cfg.EventBus, cfg.SkillExec, cb),
		callbacks:     cb,
		eventBus:      cfg.EventBus,
		reviewCreator: cfg.ReviewCreator,
		authChecker:   cfg.AuthChecker,
	}
}

// --------------------------------------------------------------------------
// Start / Stop — 生命週期管理
// --------------------------------------------------------------------------

// Start 啟動排程引擎。
//
// 流程：
//  1. 從磁碟載入所有 Job
//  2. 執行補執行（catchUp）：檢查錯過的 Job 並補跑一次
//  3. 啟動 Ticker goroutine，每分鐘掃描一次
//
// ctx 用於控制引擎的生命週期，當 ctx 被取消時引擎會停止。
func (s *Service) Start(ctx context.Context) error {
	// 載入持久化的 Job
	loaded, err := s.store.LoadJobs()
	if err != nil {
		return fmt.Errorf("scheduler: 載入排程任務失敗: %w", err)
	}

	s.mu.Lock()
	s.jobs = make([]*Job, len(loaded))
	for i := range loaded {
		job := loaded[i]
		s.jobs[i] = &job
	}
	s.mu.Unlock()

	// 補執行錯過的 Job
	s.catchUp(ctx)

	// 建立可取消的 context
	tickCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// 啟動 Ticker goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runTicker(tickCtx)
	}()

	return nil
}

// Stop 停止排程引擎，等待所有正在執行的 Job 完成。
func (s *Service) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// CancelJob 取消正在執行中的指定任務。
func (s *Service) CancelJob(id string) error {
	s.runMu.Lock()
	cancel, ok := s.running[id]
	s.runMu.Unlock()
	if !ok {
		return fmt.Errorf("scheduler: 任務 %q 目前沒有在執行", id)
	}
	cancel()
	return nil
}

// --------------------------------------------------------------------------
// Callbacks — 對外暴露回呼註冊表
// --------------------------------------------------------------------------

// Callbacks 回傳回呼函式註冊表，供外部模組註冊 callback 動作。
func (s *Service) Callbacks() *CallbackRegistry {
	return s.callbacks
}

// --------------------------------------------------------------------------
// CRUD — Job 管理方法
// --------------------------------------------------------------------------

// CreateJob 建立一個新的排程任務並儲存至磁碟。
func (s *Service) CreateJob(name, cronExpr string, actionType ActionType, actionPayload string, optional ...string) (*Job, error) {
	job, err := NewJob(name, cronExpr, actionType, actionPayload, optional...)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.jobs = append(s.jobs, job)
	err = s.persistJobsLocked()
	s.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("scheduler: 儲存任務失敗: %w", err)
	}
	return job, nil
}

// ListJobs 回傳所有排程任務的副本（不含內部指標）。
func (s *Service) ListJobs() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Job, len(s.jobs))
	for i, j := range s.jobs {
		result[i] = *j
	}
	return result
}

// DeleteJob 刪除指定 ID 的排程任務。
func (s *Service) DeleteJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findJobIndexLocked(id)
	if idx == -1 {
		return fmt.Errorf("scheduler: 找不到任務 %q", id)
	}

	// 移除
	s.jobs = append(s.jobs[:idx], s.jobs[idx+1:]...)
	return s.persistJobsLocked()
}

// PauseJob 暫停指定 ID 的排程任務（設 Enabled = false）。
func (s *Service) PauseJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findJobIndexLocked(id)
	if idx == -1 {
		return fmt.Errorf("scheduler: 找不到任務 %q", id)
	}
	s.jobs[idx].Enabled = false
	return s.persistJobsLocked()
}

// ResumeJob 恢復指定 ID 的排程任務（設 Enabled = true），並重新計算 NextFire。
func (s *Service) ResumeJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findJobIndexLocked(id)
	if idx == -1 {
		return fmt.Errorf("scheduler: 找不到任務 %q", id)
	}

	job := s.jobs[idx]
	job.Enabled = true

	// 重新計算 NextFire
	parsed, err := ParseCron(job.CronExpr)
	if err == nil {
		next := parsed.NextAfter(time.Now())
		if !next.IsZero() {
			job.NextFire = next.Format(time.RFC3339)
		}
	}

	return s.persistJobsLocked()
}

// GetJobHistory 查詢指定任務的執行歷史紀錄。
func (s *Service) GetJobHistory(jobID string, limit int) ([]JobExecution, error) {
	return s.store.GetHistory(jobID, limit)
}

// --------------------------------------------------------------------------
// runTicker — Ticker 主迴圈
// --------------------------------------------------------------------------

// runTicker 為 Ticker 的 goroutine 主體。
// 每分鐘呼叫一次 tick()，直到 ctx 被取消。
func (s *Service) runTicker(ctx context.Context) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			s.tick(ctx, t)
		}
	}
}

// --------------------------------------------------------------------------
// tick — 單次掃描邏輯
// --------------------------------------------------------------------------

// tick 在每分鐘執行一次，掃描所有已啟用的 Job 並觸發到期者。
func (s *Service) tick(ctx context.Context, now time.Time) {
	s.mu.Lock()
	// 建立快照避免長時間持鎖
	snapshot := make([]*Job, len(s.jobs))
	copy(snapshot, s.jobs)
	s.mu.Unlock()

	for _, job := range snapshot {
		if !job.Enabled {
			continue
		}

		// 檢查 NextFire 是否到期
		if job.NextFire == "" {
			continue
		}
		nextFire, err := time.Parse(time.RFC3339, job.NextFire)
		if err != nil {
			continue
		}
		if now.Before(nextFire) {
			continue
		}

		// 檢查是否正在執行（重疊保護）
		s.runMu.Lock()
		if _, running := s.running[job.ID]; running {
			s.runMu.Unlock()
			// 記錄跳過事件
			s.recordSkipped(job, now)
			continue
		}
		jobCtx, cancel := context.WithCancel(ctx)
		s.running[job.ID] = cancel
		s.runMu.Unlock()

		// 在獨立 goroutine 中執行
		s.wg.Add(1)
		go func(j *Job) {
			defer s.wg.Done()
			defer func() {
				s.runMu.Lock()
				delete(s.running, j.ID)
				s.runMu.Unlock()
			}()
			s.executeJob(jobCtx, j, now)
		}(job)
	}
}

// --------------------------------------------------------------------------
// executeJob — 執行單一 Job（含重試邏輯）
// --------------------------------------------------------------------------

// executeJob 執行指定 Job 的動作，失敗時自動重試一次。
//
// 流程：
//  1. 根據 ActionType 取得 Action 實作
//  2. 呼叫 Execute，記錄耗時
//  3. 若失敗：等 5 秒 → 重試一次
//  4. 更新 LastFired / NextFire / ConsecutiveFailures
//  5. 記錄執行歷史 + 發送事件通知
func (s *Service) executeJob(ctx context.Context, job *Job, firedAt time.Time) {
	// — Risk gate —
	if ShouldBlockExecution(job.RiskClass) {
		// Destructive+: 只建提醒，不執行
		if s.reviewCreator != nil {
			_ = s.reviewCreator.CreateSchedulerReviewCard(job.ID, job.Name, job.RiskClass, "排程到點但風險等級不允許自動執行")
		}
		s.updateNextFire(job)
		return
	}
	if ShouldRequireAuth(job.RiskClass) {
		if s.authChecker == nil || !s.authChecker.IsAuthorizedForJob(job.ID) {
			if s.reviewCreator != nil {
				_ = s.reviewCreator.CreateSchedulerReviewCard(job.ID, job.Name, job.RiskClass, "排程需要授權但無有效授權")
			}
			s.updateNextFire(job)
			return
		}
		// 有授權：繼續執行
	}
	if NeedsPayloadRecheck(job.RiskClass) {
		currentHash := computePayloadHash(job.ActionPayload)
		if job.PayloadHash != "" && currentHash != job.PayloadHash {
			if s.reviewCreator != nil {
				_ = s.reviewCreator.CreateSchedulerReviewCard(job.ID, job.Name, job.RiskClass, "排程 payload 已被修改，需重新確認")
			}
			s.updateNextFire(job)
			return
		}
	}

	// 取得對應的 Action
	action, err := s.resolver.Resolve(job.ActionType)
	if err != nil {
		s.recordFailure(job, firedAt, 0, err, false)
		return
	}

	// 第一次執行
	start := time.Now()
	execErr := action.Execute(ctx, job.ActionPayload)
	duration := time.Since(start).Milliseconds()

	// 檢查是否被使用者取消
	if ctx.Err() == context.Canceled {
		s.recordCancelled(job, firedAt, duration)
		return
	}

	if execErr != nil {
		// 等待後重試一次
		time.Sleep(retryDelay)

		start2 := time.Now()
		retryErr := action.Execute(ctx, job.ActionPayload)
		duration2 := time.Since(start2).Milliseconds()

		// 重試後再次檢查取消
		if ctx.Err() == context.Canceled {
			s.recordCancelled(job, firedAt, duration+duration2)
			return
		}

		if retryErr != nil {
			// 重試也失敗
			s.recordFailure(job, firedAt, duration+duration2, retryErr, true)
			return
		}

		// 重試成功
		s.recordSuccess(job, firedAt, duration+duration2, true)
		return
	}

	// 第一次就成功
	s.recordSuccess(job, firedAt, duration, false)
}

// --------------------------------------------------------------------------
// catchUp — 啟動時補執行
// --------------------------------------------------------------------------

// catchUp 在啟動時檢查所有已啟用的 Job，若 NextFire 已過期則補執行一次。
// 只補最近一次，不會連鎖補回所有錯過的觸發。
func (s *Service) catchUp(ctx context.Context) {
	now := time.Now()

	s.mu.Lock()
	snapshot := make([]*Job, len(s.jobs))
	copy(snapshot, s.jobs)
	s.mu.Unlock()

	for _, job := range snapshot {
		if !job.Enabled {
			continue
		}
		if job.NextFire == "" {
			continue
		}

		nextFire, err := time.Parse(time.RFC3339, job.NextFire)
		if err != nil {
			continue
		}

		// 只補曾經觸發過且已過期的 Job；新建立但尚未觸發的 Job 不補跑。
		if job.LastFired != "" && now.After(nextFire) {
			// High-risk missed job：不自動補執行，發事件由 app 層建立 CatchUpCard
			if !ShouldAutoExecute(job.RiskClass) {
				if s.eventBus != nil {
					s.eventBus.Emit(eventbus.EventSchedulerJobMissedCatchUp, map[string]string{
						"job_id":     job.ID,
						"job_name":   job.Name,
						"risk_class": job.RiskClass,
						"missed":     job.NextFire,
						"blocked":    "true",
					})
				}
				s.updateNextFire(job)
				continue
			}

			// Low/Medium：照常補執行
			if s.eventBus != nil {
				s.eventBus.Emit(eventbus.EventSchedulerJobMissedCatchUp, map[string]string{
					"job_id":     job.ID,
					"job_name":   job.Name,
					"risk_class": job.RiskClass,
					"missed":     job.NextFire,
					"blocked":    "false",
				})
			}
			s.executeJob(ctx, job, now)
		}
	}
}

// --------------------------------------------------------------------------
// 紀錄輔助方法
// --------------------------------------------------------------------------

// recordSuccess 記錄成功的執行結果，更新 Job 狀態並持久化。
func (s *Service) recordSuccess(job *Job, firedAt time.Time, durationMs int64, retried bool) {
	s.mu.Lock()
	job.LastFired = firedAt.Format(time.RFC3339)
	job.ConsecutiveFailures = 0

	// 重算 NextFire
	parsed, err := ParseCron(job.CronExpr)
	if err == nil {
		next := parsed.NextAfter(time.Now())
		if !next.IsZero() {
			job.NextFire = next.Format(time.RFC3339)
		}
	}
	_ = s.persistJobsLocked()
	s.mu.Unlock()

	// 記錄執行歷史
	exec := JobExecution{
		JobID:    job.ID,
		FiredAt:  firedAt.Format(time.RFC3339),
		Duration: durationMs,
		Status:   ExecStatusSuccess,
		Retried:  retried,
	}
	_ = s.store.AppendHistory(exec)

	// 發送事件通知
	if s.eventBus != nil {
		s.eventBus.Emit(eventbus.EventSchedulerJobFired, map[string]interface{}{
			"job_id":      job.ID,
			"job_name":    job.Name,
			"fired_at":    exec.FiredAt,
			"duration_ms": durationMs,
			"success":     true,
			"retried":     retried,
		})
	}
}

// recordFailure 記錄失敗的執行結果，更新 Job 狀態並持久化。
func (s *Service) recordFailure(job *Job, firedAt time.Time, durationMs int64, execErr error, retried bool) {
	s.mu.Lock()
	job.LastFired = firedAt.Format(time.RFC3339)
	job.ConsecutiveFailures++

	// 重算 NextFire
	parsed, err := ParseCron(job.CronExpr)
	if err == nil {
		next := parsed.NextAfter(time.Now())
		if !next.IsZero() {
			job.NextFire = next.Format(time.RFC3339)
		}
	}
	_ = s.persistJobsLocked()
	s.mu.Unlock()

	// 記錄執行歷史
	exec := JobExecution{
		JobID:    job.ID,
		FiredAt:  firedAt.Format(time.RFC3339),
		Duration: durationMs,
		Status:   ExecStatusFailed,
		Error:    execErr.Error(),
		Retried:  retried,
	}
	_ = s.store.AppendHistory(exec)

	// 發送錯誤事件通知
	if s.eventBus != nil {
		s.eventBus.Emit(eventbus.EventSchedulerJobError, map[string]interface{}{
			"job_id":               job.ID,
			"job_name":             job.Name,
			"error":                execErr.Error(),
			"consecutive_failures": job.ConsecutiveFailures,
			"retried":              retried,
		})
	}
}

// recordSkipped 記錄因重疊而跳過的執行。
func (s *Service) recordSkipped(job *Job, now time.Time) {
	exec := JobExecution{
		JobID:   job.ID,
		FiredAt: now.Format(time.RFC3339),
		Status:  ExecStatusSkipped,
	}
	_ = s.store.AppendHistory(exec)

	if s.eventBus != nil {
		s.eventBus.Emit(eventbus.EventSchedulerJobSkipped, map[string]string{
			"job_id":   job.ID,
			"job_name": job.Name,
		})
	}
}

// recordCancelled 記錄被使用者取消的執行，不影響 ConsecutiveFailures。
func (s *Service) recordCancelled(job *Job, firedAt time.Time, durationMs int64) {
	s.mu.Lock()
	job.LastFired = firedAt.Format(time.RFC3339)

	// 重算 NextFire
	parsed, err := ParseCron(job.CronExpr)
	if err == nil {
		next := parsed.NextAfter(time.Now())
		if !next.IsZero() {
			job.NextFire = next.Format(time.RFC3339)
		}
	}
	_ = s.persistJobsLocked()
	s.mu.Unlock()

	exec := JobExecution{
		JobID:    job.ID,
		FiredAt:  firedAt.Format(time.RFC3339),
		Duration: durationMs,
		Status:   ExecStatusCancelled,
	}
	_ = s.store.AppendHistory(exec)

	if s.eventBus != nil {
		s.eventBus.Emit(eventbus.EventSchedulerJobCancelled, map[string]interface{}{
			"job_id":      job.ID,
			"job_name":    job.Name,
			"fired_at":    exec.FiredAt,
			"duration_ms": durationMs,
		})
	}
}

// --------------------------------------------------------------------------
// updateNextFire — 僅更新 NextFire，不記錄執行（風險閘門擋下時使用）
// --------------------------------------------------------------------------

// updateNextFire 重算並持久化 NextFire，不影響 LastFired 或 ConsecutiveFailures。
func (s *Service) updateNextFire(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()

	parsed, err := ParseCron(job.CronExpr)
	if err == nil {
		next := parsed.NextAfter(time.Now())
		if !next.IsZero() {
			job.NextFire = next.Format(time.RFC3339)
		}
	}
	_ = s.persistJobsLocked()
}

// --------------------------------------------------------------------------
// 內部輔助方法
// --------------------------------------------------------------------------

// findJobIndexLocked 在已持鎖的情況下，找到指定 ID 的 Job 索引。
// 找不到時回傳 -1。呼叫端必須已持有 s.mu。
func (s *Service) findJobIndexLocked(id string) int {
	for i, j := range s.jobs {
		if j.ID == id {
			return i
		}
	}
	return -1
}

// persistJobsLocked 將目前的 jobs 切片持久化至磁碟。
// 呼叫端必須已持有 s.mu。
func (s *Service) persistJobsLocked() error {
	flat := make([]Job, len(s.jobs))
	for i, j := range s.jobs {
		flat[i] = *j
	}
	return s.store.SaveJobs(flat)
}
