package services

import (
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// StartDailySyncScheduler 매일 01:00 KST에 BTC/ETH 시세 싱크
func StartDailySyncScheduler() {
	// 서버 시작 시 1회 즉시 지표 계산
	go func() {
		for _, coin := range []string{"BTC", "ETH", "SOL"} {
			if err := CalcAndStoreAllIndicators(coin); err != nil {
				log.Printf("[sync-scheduler] 초기 지표 계산 실패 %s: %v", coin, err)
			}
		}
	}()

	go func() {
		for {
			now := time.Now().In(time.FixedZone("KST", 9*60*60))
			// 다음 01:00 KST 계산
			next := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())
			if !now.Before(next) {
				next = next.Add(24 * time.Hour)
			}
			waitDur := next.Sub(now)
			log.Printf("[sync-scheduler] 다음 싱크까지 %.1f시간", waitDur.Hours())
			time.Sleep(waitDur)

			// 싱크 실행
			runSync()
		}
	}()
}

func runSync() {
	coins := []string{"BTC", "ETH", "SOL"}
	for _, coin := range coins {
		url := "http://localhost:8080/api/v1/sync"
		body := strings.NewReader(`{"coin":"` + coin + `"}`)
		resp, err := http.Post(url, "application/json", body)
		if err != nil {
			log.Printf("[sync-scheduler] %s 싱크 실패: %v", coin, err)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		log.Printf("[sync-scheduler] %s 싱크 완료 (status: %d)", coin, resp.StatusCode)
	}

	// 싱크 후 지표 계산
	for _, coin := range coins {
		if err := CalcAndStoreAllIndicators(coin); err != nil {
			log.Printf("[sync-scheduler] %s 지표 계산 실패: %v", coin, err)
		}
	}
}
