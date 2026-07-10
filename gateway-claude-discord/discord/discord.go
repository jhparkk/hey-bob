package discord

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)


const (
	maxMsgLen = 2000
	resetCmd  = "!reset"
	cronCmd   = "!cron"
)

func (h *Handler) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		log.Printf("[discord] failed to get channel %s: %v", m.ChannelID, err)
		return
	}

	isDM := channel.Type == discordgo.ChannelTypeDM
	isMentioned := isBotMentioned(s.State.User.ID, m.Mentions)

	// 봇이 포함된 채널 또는 DM이면 모두 응답 (멘션 불필요)
	_ = isDM

	content := extractContent(s.State.User.ID, m.Content, isMentioned)
	if content == "" {
		s.ChannelMessageSend(m.ChannelID, "메시지를 입력해주세요.")
		return
	}

	// Session reset command
	if strings.EqualFold(content, resetCmd) {
		if err := h.sessionHandler.ResetSession(m.ChannelID); err != nil {
			log.Printf("[discord] reset session error: %v", err)
			s.ChannelMessageSend(m.ChannelID, "세션 초기화 중 오류가 발생했습니다.")
			return
		}
		s.ChannelMessageSend(m.ChannelID, "세션이 초기화되었습니다. 새 대화를 시작합니다.")
		return
	}

	// Cron commands: !cron list | !cron run <job-name>
	if strings.HasPrefix(strings.ToLower(content), cronCmd) {
		h.handleCronCommand(s, m.ChannelID, strings.TrimSpace(content[len(cronCmd):]))
		return
	}

	// Send typing indicator while Claude is working
	stopTyping := startTyping(s, m.ChannelID)

	log.Printf("[discord] channel=%s user=%s message=%q", m.ChannelID, m.Author.Username, truncate(content, 80))

	response, err := h.sessionHandler.SendMessage(m.ChannelID, content)
	stopTyping()

	if err != nil {
		log.Printf("[discord] claude error for channel %s: %v", m.ChannelID, err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Claude 오류: %v", err))
		return
	}

	log.Printf("[discord] channel=%s response=%q", m.ChannelID, truncate(response, 80))

	if h.memory != nil {
		h.memory.Append(m.Author.Username, content, response, time.Now())
	}

	sendChunked(s, m.ChannelID, response)
}

func isBotMentioned(botID string, mentions []*discordgo.User) bool {
	for _, u := range mentions {
		if u.ID == botID {
			return true
		}
	}
	return false
}

func extractContent(botID, content string, isMentioned bool) string {
	if isMentioned {
		content = strings.ReplaceAll(content, fmt.Sprintf("<@%s>", botID), "")
		content = strings.ReplaceAll(content, fmt.Sprintf("<@!%s>", botID), "")
	}
	return strings.TrimSpace(content)
}

// startTyping sends a typing indicator every 8 seconds and returns a stop function.
func startTyping(s *discordgo.Session, channelID string) func() {
	stop := make(chan struct{})
	var once sync.Once
	go func() {
		s.ChannelTyping(channelID)
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				s.ChannelTyping(channelID)
			}
		}
	}()
	return func() { once.Do(func() { close(stop) }) }
}

// sendChunked splits content at newline boundaries to stay under Discord's 2000-char limit.
func sendChunked(s *discordgo.Session, channelID, content string) {
	if content == "" {
		return
	}

	for len(content) > 0 {
		if len(content) <= maxMsgLen {
			s.ChannelMessageSend(channelID, content)
			return
		}

		// Find last newline within the limit
		cut := strings.LastIndex(content[:maxMsgLen], "\n")
		if cut <= 0 {
			cut = maxMsgLen
		}

		if _, err := s.ChannelMessageSend(channelID, content[:cut]); err != nil {
			log.Printf("[discord] send error: %v", err)
			return
		}
		content = strings.TrimPrefix(content[cut:], "\n")
	}
}

func (h *Handler) handleCronCommand(s *discordgo.Session, channelID, args string) {
	if h.cron == nil {
		s.ChannelMessageSend(channelID, "cron이 활성화되지 않았습니다.")
		return
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		s.ChannelMessageSend(channelID, "사용법: `!cron list` | `!cron run <job-name>`")
		return
	}

	switch parts[0] {
	case "list":
		names := h.cron.List()
		if len(names) == 0 {
			s.ChannelMessageSend(channelID, "등록된 cron job이 없습니다.")
			return
		}
		s.ChannelMessageSend(channelID, "**등록된 cron jobs:**\n```\n"+strings.Join(names, "\n")+"\n```")

	case "run":
		if len(parts) < 2 {
			s.ChannelMessageSend(channelID, "사용법: `!cron run <job-name>`")
			return
		}
		name := strings.Join(parts[1:], " ")
		if err := h.cron.Trigger(name); err != nil {
			s.ChannelMessageSend(channelID, fmt.Sprintf("실행 실패: %v", err))
			return
		}
		s.ChannelMessageSend(channelID, fmt.Sprintf("⚡ `%s` 실행 시작 (백그라운드)", name))

	case "enable", "disable":
		if len(parts) < 2 {
			s.ChannelMessageSend(channelID, fmt.Sprintf("사용법: `!cron %s <job-name>`", parts[0]))
			return
		}
		name := strings.Join(parts[1:], " ")
		enabled := parts[0] == "enable"
		if h.cron.SetEnabled == nil {
			s.ChannelMessageSend(channelID, "enable/disable이 지원되지 않습니다.")
			return
		}
		if err := h.cron.SetEnabled(name, enabled); err != nil {
			s.ChannelMessageSend(channelID, fmt.Sprintf("변경 실패: %v", err))
			return
		}
		icon := "✅"
		if !enabled {
			icon = "❌"
		}
		s.ChannelMessageSend(channelID, fmt.Sprintf("%s `%s` %s됨 (crontab 반영 완료)", icon, name, parts[0]))

	case "reload":
		if h.cron.Reload == nil {
			s.ChannelMessageSend(channelID, "reload가 지원되지 않습니다.")
			return
		}
		if err := h.cron.Reload(); err != nil {
			s.ChannelMessageSend(channelID, fmt.Sprintf("reload 실패: %v", err))
			return
		}
		s.ChannelMessageSend(channelID, "🔄 jobs.json 재로드 및 crontab 반영 완료")

	default:
		s.ChannelMessageSend(channelID, "사용법: `!cron list` | `!cron run <name>` | `!cron enable <name>` | `!cron disable <name>` | `!cron reload`")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
