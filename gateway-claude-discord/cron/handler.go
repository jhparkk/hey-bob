package cron

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gateway-claude-discord/session"

	"github.com/bwmarrin/discordgo"
	robfigcron "github.com/robfig/cron/v3"
)

const defaultTimeout = 5 * time.Minute

type Handler struct {
	session       *session.Handler
	discord       *discordgo.Session
	jobs          []Job
	scheduledJobs []Job // reschedule() 시점의 스냅샷
	jobsMu        sync.Mutex
	jobsPath      string
	scheduler     *robfigcron.Cron
}

func NewHandler(sh *session.Handler, dg *discordgo.Session) *Handler {
	return &Handler{session: sh, discord: dg}
}

// StartScheduler는 jobs.json 기반으로 내부 스케줄러를 시작한다.
func (h *Handler) StartScheduler() {
	h.jobsMu.Lock()
	defer h.jobsMu.Unlock()
	h.reschedule()
}

// StopScheduler는 내부 스케줄러를 중지한다.
func (h *Handler) StopScheduler() {
	h.jobsMu.Lock()
	defer h.jobsMu.Unlock()
	if h.scheduler != nil {
		h.scheduler.Stop()
		h.scheduler = nil
	}
}

// reschedule은 현재 h.jobs 상태로 스케줄러를 재시작한다. jobsMu를 보유한 채 호출해야 한다.
func (h *Handler) reschedule() {
	if h.scheduler != nil {
		h.scheduler.Stop()
	}

	c := robfigcron.New()
	for _, job := range h.jobs {
		if !job.Enabled {
			continue
		}
		j := job
		tz := j.Schedule.TZ
		if tz == "" {
			tz = "UTC"
		}
		spec := fmt.Sprintf("CRON_TZ=%s %s", tz, j.Schedule.Expr)
		if _, err := c.AddFunc(spec, func() { h.runJob(j) }); err != nil {
			log.Printf("[cron] invalid schedule for %s (%s): %v", j.Name, spec, err)
		}
	}
	c.Start()
	h.scheduler = c
	h.scheduledJobs = append([]Job{}, h.jobs...)
	log.Printf("[cron] scheduler restarted with %d enabled jobs", len(h.enabledCount()))
}

func (h *Handler) enabledCount() []Job {
	var out []Job
	for _, j := range h.jobs {
		if j.Enabled {
			out = append(out, j)
		}
	}
	return out
}

func (h *Handler) runJob(job Job) {
	log.Printf("[cron] job %s: starting", job.Name)

	if job.Schedule.StaggerMs > 0 {
		delay := time.Duration(rand.Int63n(job.Schedule.StaggerMs)) * time.Millisecond
		log.Printf("[cron] job %s: stagger delay %v", job.Name, delay)
		time.Sleep(delay)
	}

	timeout := defaultTimeout
	if job.Payload.TimeoutSeconds > 0 {
		timeout = time.Duration(job.Payload.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	msg := job.Payload.Message
	if job.Payload.PreprocessScript != "" {
		data, serr := runScript(ctx, job.Payload.PreprocessScript)
		if serr != nil {
			log.Printf("[cron] job %s: preprocess script error: %v", job.Name, serr)
		} else {
			msg = msg + "\n\n## 수집된 시장 데이터\n" + data
		}
	}

	response, err := h.session.RunIsolated(ctx, msg, job.Payload.Model, job.Payload.WorkDir)
	if err != nil {
		log.Printf("[cron] job %s: error: %v", job.Name, err)
		return
	}

	log.Printf("[cron] job %s: done (%d chars)", job.Name, len(response))

	if job.Delivery.Mode == "announce" && response != "" {
		if job.Delivery.NotifyOnlyIf != "" && !strings.Contains(response, job.Delivery.NotifyOnlyIf) {
			log.Printf("[cron] job %s: notifyOnlyIf %q not found — delivery suppressed", job.Name, job.Delivery.NotifyOnlyIf)
			return
		}
		if isQuietHour(job.Delivery.QuietHours, job.Schedule.TZ) {
			log.Printf("[cron] job %s: quiet hours — delivery suppressed", job.Name)
			return
		}
		h.deliver(job.Name, job.Delivery.To, response)
	}
}

func (h *Handler) deliver(jobName, to, content string) {
	parts := strings.SplitN(to, ":", 2)
	if len(parts) != 2 {
		log.Printf("[cron] job %s: invalid delivery target: %s", jobName, to)
		return
	}

	kind, id := parts[0], parts[1]

	var channelID string
	switch kind {
	case "user":
		ch, err := h.discord.UserChannelCreate(id)
		if err != nil {
			log.Printf("[cron] job %s: DM create failed: %v", jobName, err)
			return
		}
		channelID = ch.ID
	case "channel":
		channelID = id
	default:
		log.Printf("[cron] job %s: unknown delivery kind: %s", jobName, kind)
		return
	}

	sendChunked(h.discord, channelID, content)
	log.Printf("[cron] job %s: delivered to %s", jobName, to)
}

func runScript(ctx context.Context, scriptPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func sendChunked(s *discordgo.Session, channelID, content string) {
	const maxLen = 2000
	for len(content) > 0 {
		if len(content) <= maxLen {
			s.ChannelMessageSend(channelID, content)
			return
		}
		cut := strings.LastIndex(content[:maxLen], "\n")
		if cut <= 0 {
			cut = maxLen
		}
		if _, err := s.ChannelMessageSend(channelID, content[:cut]); err != nil {
			log.Printf("[cron] send chunk error: %v", err)
			return
		}
		content = strings.TrimPrefix(content[cut:], "\n")
	}
}
