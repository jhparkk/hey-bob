package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// LoadJobs는 jobs.json을 읽어 Handler에 등록한다.
func (h *Handler) LoadJobs(jobsPath string) error {
	data, err := os.ReadFile(jobsPath)
	if err != nil {
		return fmt.Errorf("read jobs file: %w", err)
	}

	var jf JobsFile
	if err := json.Unmarshal(data, &jf); err != nil {
		return fmt.Errorf("parse jobs file: %w", err)
	}

	h.jobsMu.Lock()
	h.jobs = jf.Jobs
	h.jobsPath = jobsPath
	h.jobsMu.Unlock()
	return nil
}

// Reload는 jobs.json을 다시 읽어 메모리와 스케줄러를 갱신한다.
func (h *Handler) Reload() error {
	if err := h.LoadJobs(h.jobsPath); err != nil {
		return err
	}
	h.jobsMu.Lock()
	h.reschedule()
	h.jobsMu.Unlock()
	return nil
}

// SetEnabled는 job의 enabled 상태를 변경하고 jobs.json과 스케줄러를 갱신한다.
func (h *Handler) SetEnabled(name string, enabled bool) error {
	h.refreshJobs()
	h.jobsMu.Lock()
	defer h.jobsMu.Unlock()

	found := false
	for i := range h.jobs {
		if h.jobs[i].Name == name {
			h.jobs[i].Enabled = enabled
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("job not found: %q", name)
	}

	if err := h.saveJobs(); err != nil {
		return err
	}

	h.reschedule()
	return nil
}

func (h *Handler) saveJobs() error {
	if h.jobsPath == "" {
		return nil
	}
	jf := JobsFile{Version: 1, Jobs: h.jobs}
	data, err := json.MarshalIndent(jf, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.jobsPath, data, 0644)
}

// refreshJobs는 jobsPath가 있으면 파일을 다시 읽어 in-memory 상태를 갱신한다 (스케줄러 재시작 없음).
func (h *Handler) refreshJobs() {
	if h.jobsPath == "" {
		return
	}
	data, err := os.ReadFile(h.jobsPath)
	if err != nil {
		return
	}
	var jf JobsFile
	if err := json.Unmarshal(data, &jf); err != nil {
		return
	}
	h.jobsMu.Lock()
	h.jobs = jf.Jobs
	h.jobsMu.Unlock()
}

func (h *Handler) findJob(name string) (Job, bool) {
	for _, j := range h.jobs {
		if j.Name == name {
			return j, true
		}
	}
	return Job{}, false
}

// Jobs는 실제 스케줄러에 등록된 잡 목록을 반환한다 (마지막 reschedule 시점 기준).
func (h *Handler) Jobs() []Job {
	h.jobsMu.Lock()
	defer h.jobsMu.Unlock()
	return append([]Job{}, h.scheduledJobs...)
}

// Trigger는 이름으로 잡을 즉시 실행한다 (Discord !cron run용). 최신 파일 상태를 반영한다.
func (h *Handler) Trigger(name string) error {
	h.refreshJobs()
	job, ok := h.findJob(name)
	if !ok {
		return fmt.Errorf("job not found: %q", name)
	}
	go h.runJob(job)
	return nil
}

// isQuietHour는 현재 시각이 qh의 억제 범위 안인지 확인한다.
// qh가 nil이면 항상 false(발송 허용).
func isQuietHour(qh *QuietHours, scheduleTZ string) bool {
	if qh == nil || qh.From == "" || qh.To == "" {
		return false
	}
	tz := qh.TZ
	if tz == "" {
		tz = scheduleTZ
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	fromMin := parseHHMM(qh.From)
	toMin := parseHHMM(qh.To)
	nowMin := now.Hour()*60 + now.Minute()

	if fromMin <= toMin {
		// 같은 날 범위 (e.g. 09:00~18:00)
		return nowMin >= fromMin && nowMin < toMin
	}
	// 자정을 넘는 범위 (e.g. 23:00~07:00)
	return nowMin >= fromMin || nowMin < toMin
}

func parseHHMM(s string) int {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m
}
