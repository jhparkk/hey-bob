package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxRecent = 20

// Writer appends conversation turns to daily .md files and current-memory.md.
type Writer struct {
	dir string
}

func NewWriter(dir string) *Writer {
	_ = os.MkdirAll(dir, 0755)
	return &Writer{dir: dir}
}

// Append records one conversation turn (fire-and-forget; errors are logged only).
func (w *Writer) Append(username, userMsg, reply string, ts time.Time) {
	if err := w.appendDaily(username, userMsg, reply, ts); err != nil {
		fmt.Fprintf(os.Stderr, "[memory] daily write error: %v\n", err)
	}
	if err := w.updateCurrent(username, userMsg, reply, ts); err != nil {
		fmt.Fprintf(os.Stderr, "[memory] current-memory write error: %v\n", err)
	}
}

// ─── daily file ───────────────────────────────────────────────────────────────

func (w *Writer) appendDaily(username, userMsg, reply string, ts time.Time) error {
	path := filepath.Join(w.dir, ts.Format("2006-01-02")+".md")

	var header string
	if _, err := os.Stat(path); os.IsNotExist(err) {
		header = fmt.Sprintf("# %s 대화 기록\n\n", ts.Format("2006-01-02"))
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s## %s · %s\n\n**요청:** %s\n\n**응답:**\n\n%s\n\n---\n\n",
		header, ts.Format("15:04:05"), username, userMsg, reply)
	return err
}

// ─── current-memory.md ────────────────────────────────────────────────────────

const (
	entryBeginPrefix = "<!-- ENTRY "
	entryBeginSuffix = " -->"
	entryEnd         = "<!-- /ENTRY -->"
)

type entry struct {
	ts       time.Time
	username string
	userMsg  string
	reply    string
}

func (e entry) archiveLine() string {
	return fmt.Sprintf("- `%s` **%s:** %s → %s",
		e.ts.Format("2006-01-02 15:04"), e.username,
		truncStr(e.userMsg, 60), truncStr(e.reply, 100))
}

func entryBlock(e entry) string {
	marker := fmt.Sprintf("%s%s %s%s", entryBeginPrefix, e.ts.UTC().Format(time.RFC3339), e.username, entryBeginSuffix)
	body := fmt.Sprintf("### %s · %s\n\n**요청:** %s\n\n**응답:**\n\n%s\n\n",
		e.ts.Format("2006-01-02 15:04:05"), e.username, e.userMsg, e.reply)
	return marker + "\n" + body + entryEnd + "\n\n---\n\n"
}

func (w *Writer) updateCurrent(username, userMsg, reply string, ts time.Time) error {
	path := filepath.Join(w.dir, "current-memory.md")

	raw, _ := os.ReadFile(path)
	archive, entries := parseCurrent(string(raw))

	newEntry := entry{ts: ts, username: username, userMsg: userMsg, reply: reply}

	if len(entries) >= maxRecent {
		archive = append(archive, entries[0].archiveLine())
		entries = entries[1:]
	}
	entries = append(entries, newEntry)

	return os.WriteFile(path, []byte(renderCurrent(archive, entries)), 0644)
}

func parseCurrent(content string) (archive []string, entries []entry) {
	// Parse archive lines between "## 아카이브" and "---"
	if ai := strings.Index(content, "## 아카이브\n\n"); ai != -1 {
		after := content[ai+len("## 아카이브\n\n"):]
		end := strings.Index(after, "\n---\n")
		if end == -1 {
			end = len(after)
		}
		for _, line := range strings.Split(after[:end], "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "- ") {
				archive = append(archive, strings.TrimSpace(line))
			}
		}
	}

	// Parse entries by HTML comment markers
	remaining := content
	for {
		start := strings.Index(remaining, entryBeginPrefix)
		if start == -1 {
			break
		}
		endTag := strings.Index(remaining[start:], entryEnd)
		if endTag == -1 {
			break
		}

		block := remaining[start : start+endTag+len(entryEnd)]

		// Extract timestamp + username from marker line
		markerClose := strings.Index(block, entryBeginSuffix)
		if markerClose == -1 {
			remaining = remaining[start+1:]
			continue
		}
		meta := block[len(entryBeginPrefix):markerClose]
		parts := strings.SplitN(meta, " ", 2)

		var e entry
		if len(parts) == 2 {
			e.ts, _ = time.Parse(time.RFC3339, parts[0])
			e.username = parts[1]
		}

		// Extract userMsg and reply from body
		body := block[markerClose+len(entryBeginSuffix):]
		body = strings.TrimSuffix(strings.TrimSpace(body), entryEnd)
		if idx := strings.Index(body, "**요청:** "); idx != -1 {
			after := body[idx+len("**요청:** "):]
			if idx2 := strings.Index(after, "\n\n**응답:**\n\n"); idx2 != -1 {
				e.userMsg = strings.TrimSpace(after[:idx2])
				e.reply = strings.TrimSpace(after[idx2+len("\n\n**응답:**\n\n"):])
			}
		}

		entries = append(entries, e)
		remaining = remaining[start+endTag+len(entryEnd):]
	}

	return
}

func renderCurrent(archive []string, entries []entry) string {
	var sb strings.Builder
	sb.WriteString("# 대화 히스토리\n\n")

	sb.WriteString("## 아카이브\n\n")
	if len(archive) == 0 {
		sb.WriteString("(없음)\n\n")
	} else {
		for _, line := range archive {
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n## 최근 대화 (최대 20건)\n\n")
	for _, e := range entries {
		sb.WriteString(entryBlock(e))
	}

	return sb.String()
}

func truncStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
