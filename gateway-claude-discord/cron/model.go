package cron

type JobsFile struct {
	Version int   `json:"version"`
	Jobs    []Job `json:"jobs"`
}

type Job struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Schedule    Schedule `json:"schedule"`
	Payload     Payload  `json:"payload"`
	Delivery    Delivery `json:"delivery"`
}

type Schedule struct {
	Kind      string `json:"kind"`
	Expr      string `json:"expr"`
	TZ        string `json:"tz"`
	StaggerMs int64  `json:"staggerMs"`
}

type Payload struct {
	Kind             string `json:"kind"`
	Message          string `json:"message"`
	TimeoutSeconds   int    `json:"timeoutSeconds"`
	PreprocessScript string `json:"preprocessScript,omitempty"`
	Model            string `json:"model,omitempty"`   // e.g. "claude-haiku-4-5", 생략 시 claude CLI 기본값
	WorkDir          string `json:"workDir,omitempty"` // claude 실행 디렉토리, 생략 시 기본 workDir()
}

type Delivery struct {
	Mode         string      `json:"mode"`                   // "announce" | "none"
	To           string      `json:"to"`                     // "user:<id>" | "channel:<id>"
	NotifyOnlyIf string      `json:"notifyOnlyIf,omitempty"` // 응답에 이 문자열 포함 시에만 전달
	QuietHours   *QuietHours `json:"quietHours,omitempty"`   // 발송 억제 시간대
}

// QuietHours는 Discord 메시지 발송을 억제할 시간 범위를 정의한다.
// From/To는 "HH:MM" (24h) 형식. 자정을 넘는 범위 지원 (e.g. "23:00"~"07:00").
type QuietHours struct {
	From string `json:"from"` // "HH:MM"
	To   string `json:"to"`   // "HH:MM"
	TZ   string `json:"tz,omitempty"` // 생략 시 schedule.tz 사용
}