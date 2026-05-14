package session

// persistentRunner keeps one claude process alive per channel.
// 메시지를 stdin으로 보내고 stdout을 읽어 응답을 수신한다.
// 응답 완료는 1초간 출력이 없을 때 감지한다(silence detection).
// TERM=dumb + NO_COLOR=1 로 TUI 렌더링을 억제해 ANSI 노이즈를 최소화한다.

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

var ansiEscape = regexp.MustCompile(`\x1b(\[[0-9;?]*[a-zA-Z]|[()][AB012]|[=<>MF78hl]|\][^\x07]*\x07)`)

const (
	startupWait     = 8 * time.Second // claude 초기화 고정 대기
	responseSilence = 1 * time.Second // 응답 완료 감지 기준(침묵)
)

type persistentRunner struct {
	ptmx *os.File
	cmd  *exec.Cmd
	mu   sync.Mutex
}

func newPersistentRunner() (*persistentRunner, error) {
	cmd := exec.Command("claude", "--dangerously-skip-permissions")
	cmd.Dir = workDir()
	// TUI/색상 출력 억제 → ANSI 노이즈 감소 및 침묵 감지 안정화
	cmd.Env = append(os.Environ(), "TERM=dumb", "NO_COLOR=1")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("pty start: %w", err)
	}

	// 터미널 크기 설정 (줄바꿈 제어)
	pty.Setsize(ptmx, &pty.Winsize{Rows: 50, Cols: 220})

	r := &persistentRunner{ptmx: ptmx, cmd: cmd}

	// 시작 배너를 고정 시간 동안 드레인
	log.Printf("[session] claude process started, draining startup (%v)...", startupWait)
	r.drainFor(startupWait)
	log.Printf("[session] startup drain complete, ready")

	return r, nil
}

func (r *persistentRunner) send(ctx context.Context, message string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := fmt.Fprintln(r.ptmx, message); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}

	raw, err := r.readUntilSilent(ctx)
	if err != nil {
		return "", err
	}

	return cleanOutput(message, raw), nil
}

// drainFor는 duration 동안 출력을 읽어 버린다.
func (r *persistentRunner) drainFor(duration time.Duration) {
	buf := make([]byte, 4096)
	r.ptmx.SetReadDeadline(time.Now().Add(duration))
	for {
		_, err := r.ptmx.Read(buf)
		if err != nil {
			break
		}
	}
	r.ptmx.SetReadDeadline(time.Time{})
}

// readUntilSilent은 responseSilence 동안 출력이 없을 때 응답 완료로 판단한다.
func (r *persistentRunner) readUntilSilent(ctx context.Context) (string, error) {
	var sb strings.Builder
	buf := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		r.ptmx.SetReadDeadline(time.Now().Add(responseSilence))
		n, err := r.ptmx.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if err != nil {
			break // 1초 침묵 = 응답 완료
		}
	}

	r.ptmx.SetReadDeadline(time.Time{})
	return sb.String(), nil
}

func (r *persistentRunner) isAlive() bool {
	if r.cmd.Process == nil {
		return false
	}
	return r.cmd.Process.Signal(syscall.Signal(0)) == nil
}

func (r *persistentRunner) close() {
	r.ptmx.Close()
	if r.cmd.Process != nil {
		r.cmd.Process.Kill()
	}
}

// cleanOutput은 ANSI 코드, 에코된 입력, 빈 프롬프트 라인을 제거한다.
func cleanOutput(input, raw string) string {
	clean := ansiEscape.ReplaceAllString(raw, "")
	clean = strings.ReplaceAll(clean, "\r\n", "\n")
	clean = strings.ReplaceAll(clean, "\r", "\n")

	lines := strings.Split(clean, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == strings.TrimSpace(input) {
			continue // 에코된 입력 제거
		}
		result = append(result, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}
