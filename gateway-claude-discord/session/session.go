package session

import (
	"context"
	"os"
	"time"
)

const claudeTimeout = 20 * time.Minute

// runner is implemented by both resumeRunner and persistentRunner.
type runner interface {
	send(ctx context.Context, message string) (string, error)
	close()
}

func workDir() string {
	if d := os.Getenv("CLAUDE_WORK_DIR"); d != "" {
		return d
	}
	return "/home/jhpark/hey-bob"
}
