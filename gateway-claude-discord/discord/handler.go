package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"gateway-claude-discord/session"
)

// CronOps는 discord 패키지가 cron 패키지에 의존하지 않도록 인터페이스로 분리한다.
type CronOps struct {
	Trigger    func(name string) error         // 잡 즉시 실행
	List       func() []string                 // 잡 이름 목록
	SetEnabled func(name string, v bool) error // 활성화/비활성화
	Reload     func() error                    // jobs.json 다시 로드
}

type Handler struct {
	token          string
	dg             *discordgo.Session
	sessionHandler *session.Handler
	cron           *CronOps // optional
}

func NewHandler(token string, sessionHandler *session.Handler) *Handler {
	return &Handler{
		token:          token,
		sessionHandler: sessionHandler,
	}
}

func (h *Handler) Start() error {
	dg, err := discordgo.New("Bot " + h.token)
	if err != nil {
		return err
	}

	h.dg = dg
	dg.AddHandler(h.onMessageCreate)

	// MESSAGE_CONTENT is a privileged intent — must be enabled in the Discord Developer Portal
	dg.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentMessageContent

	if err := dg.Open(); err != nil {
		return err
	}

	log.Printf("[discord] logged in as %s#%s", dg.State.User.Username, dg.State.User.Discriminator)
	return nil
}

func (h *Handler) Stop() {
	if h.dg != nil {
		h.dg.Close()
	}
}

// SetCron registers cron operations for Discord commands (!cron run / !cron list).
func (h *Handler) SetCron(ops *CronOps) {
	h.cron = ops
}

// Session returns the underlying discordgo session (for cron delivery).
func (h *Handler) Session() *discordgo.Session {
	return h.dg
}
