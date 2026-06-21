package session

// resumeRunner runs `claude --print --resume <id>` per message.
// 프로세스는 메시지마다 새로 시작되지만 --resume으로 대화 컨텍스트를 유지한다.
// prompt caching 덕분에 히스토리 토큰은 10% 비용(cache_read)으로 처리된다.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type claudeResult struct {
	IsError   bool   `json:"is_error"`
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
}

type resumeRunner struct {
	claudeSessionID    string
	appendSystemPrompt string // CLI 레벨 시스템 프롬프트 주입
	model              string // --model 플래그 (빈 값이면 claude 기본값)
	dir                string // cmd.Dir 오버라이드 (빈 값이면 workDir())
}

func newResumeRunner(savedSessionID string) *resumeRunner {
	return &resumeRunner{claudeSessionID: savedSessionID}
}

func (r *resumeRunner) send(ctx context.Context, message string) (string, error) {
	args := []string{
		"--print",
		"--output-format", "json",
		"--dangerously-skip-permissions",
	}
	if r.claudeSessionID != "" {
		args = append(args, "--resume", r.claudeSessionID)
	}
	if r.appendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", r.appendSystemPrompt)
	}
	if r.model != "" {
		args = append(args, "--model", r.model)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	if r.dir != "" {
		cmd.Dir = r.dir
	} else {
		cmd.Dir = workDir()
	}
	cmd.Stdin = strings.NewReader(message)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	var result claudeResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		if runErr != nil {
			errMsg := runErr.Error()
			if s := strings.TrimSpace(stderr.String()); s != "" {
				errMsg += "\nstderr: " + s
			}
			return "", fmt.Errorf("%s", errMsg)
		}
		return strings.TrimSpace(stdout.String()), nil
	}
	if result.IsError {
		return "", fmt.Errorf("claude: %s", result.Result)
	}
	if result.SessionID != "" {
		r.claudeSessionID = result.SessionID
	}
	return result.Result, nil
}

func (r *resumeRunner) sessionID() string { return r.claudeSessionID }

func (r *resumeRunner) close() {}
