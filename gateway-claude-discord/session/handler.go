package session

import (
	"context"
	"database/sql"
	"log"
	"sync"
)

type Handler struct {
	model   *Model
	runners sync.Map // channelID -> *resumeRunner
	mu      sync.Map // channelID -> *sync.Mutex
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{model: NewModel(db)}
}

func (h *Handler) channelMu(channelID string) *sync.Mutex {
	v, _ := h.mu.LoadOrStore(channelID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func (h *Handler) getOrCreate(channelID string) *resumeRunner {
	if v, ok := h.runners.Load(channelID); ok {
		return v.(*resumeRunner)
	}
	// DB에서 이전 session ID 복원 (재시작 후에도 대화 유지)
	savedID := ""
	if sess, err := h.model.GetByChannelID(channelID); err == nil && sess != nil {
		savedID = sess.ClaudeSessionID
	}
	r := newResumeRunner(savedID)
	h.runners.Store(channelID, r)
	return r
}

// SendMessage relays a message to Claude and returns the response.
func (h *Handler) SendMessage(channelID, message string) (string, error) {
	mu := h.channelMu(channelID)
	mu.Lock()
	defer mu.Unlock()

	r := h.getOrCreate(channelID)

	ctx, cancel := context.WithTimeout(context.Background(), claudeTimeout)
	defer cancel()

	response, err := r.send(ctx, message)
	if err != nil && r.claudeSessionID != "" {
		// 세션이 만료됐을 수 있으므로 초기화 후 재시도
		log.Printf("[session] channel %s: session error, retrying fresh (err: %v)", channelID, err)
		r.claudeSessionID = ""
		h.model.Delete(channelID)
		response, err = r.send(ctx, message)
	}
	if err != nil {
		return "", err
	}

	// session ID를 DB에 저장 (재시작 후 복원용)
	if r.claudeSessionID != "" {
		if dbErr := h.model.Upsert(channelID, r.claudeSessionID); dbErr != nil {
			log.Printf("[session] channel %s: failed to save session id: %v", channelID, dbErr)
		}
	}

	return response, nil
}

const localSystemPrompt = `실행 환경: WSL Ubuntu 로컬 서버. claude CLI가 직접 실행 중.
- localhost / 127.0.0.1 완전히 접근 가능 (Anthropic 클라우드가 아님)
- Bash 도구로 curl 등 로컬 명령 직접 실행 가능
- Discord 메시지 전송은 gateway가 처리 — 메시지 전송 도구 호출 불필요`

// RunIsolated runs Claude with a fresh session (no resume) and returns the response.
// cron job의 sessionTarget: "isolated" 에 사용된다.
func (h *Handler) RunIsolated(ctx context.Context, message, model, dir string) (string, error) {
	r := newResumeRunner("")
	r.appendSystemPrompt = localSystemPrompt
	r.model = model
	r.dir = dir
	return r.send(ctx, message)
}

// ResetSession clears the claude session for a channel.
func (h *Handler) ResetSession(channelID string) error {
	mu := h.channelMu(channelID)
	mu.Lock()
	defer mu.Unlock()

	h.runners.Delete(channelID)
	return h.model.Delete(channelID)
}
