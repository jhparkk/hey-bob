package main

import (
	"flag"
	"log"
	"path/filepath"
	"runtime"

	"bob-crypto-pilot/db"
	"bob-crypto-pilot/services"
)

func main() {
	dataFlag := flag.String("data", "", "data directory path (default: project root/data)")
	flag.Parse()

	var dataDir string
	if *dataFlag != "" {
		dataDir = *dataFlag
	} else {
		_, filename, _, _ := runtime.Caller(0)
		projectRoot := filepath.Join(filepath.Dir(filename), "../..")
		dataDir = filepath.Join(projectRoot, "data")
	}

	if err := db.Init(dataDir); err != nil {
		log.Fatalf("DB init failed: %v", err)
	}

	for _, coin := range []string{"BTC", "ETH", "SOL"} {
		count, err := services.FetchAndStore(coin)
		if err != nil {
			log.Printf("[backfill] %s 실패: %v", coin, err)
			continue
		}
		log.Printf("[backfill] %s 완료: %d건 upsert", coin, count)
	}
}
