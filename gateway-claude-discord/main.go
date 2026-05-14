package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	cronpkg "gateway-claude-discord/cron"
	"gateway-claude-discord/db"
	"gateway-claude-discord/discord"
	"gateway-claude-discord/session"
)

func main() {
	_ = godotenv.Load()

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN is not set")
	}

	database, err := db.Init("./gateway.db")
	if err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer database.Close()

	sessionHandler := session.NewHandler(database)
	discordHandler := discord.NewHandler(token, sessionHandler)

	if err := discordHandler.Start(); err != nil {
		log.Fatalf("discord start: %v", err)
	}
	defer discordHandler.Stop()

	// cron 설정 (jobs.json이 있을 때만)
	const jobsPath = "./cron/jobs.json"
	if _, err := os.Stat(jobsPath); err == nil {
		cronHandler := cronpkg.NewHandler(sessionHandler, discordHandler.Session())
		if err := cronHandler.LoadJobs(jobsPath); err != nil {
			log.Printf("[cron] load failed: %v", err)
		} else {
			cronHandler.StartScheduler()
			defer cronHandler.StopScheduler()
			log.Printf("[cron] scheduler started (%d jobs loaded)", len(cronHandler.Jobs()))

			discordHandler.SetCron(&discord.CronOps{
				Trigger:    cronHandler.Trigger,
				SetEnabled: cronHandler.SetEnabled,
				Reload:     cronHandler.Reload,
				List: func() []string {
					jobs := cronHandler.Jobs()
					names := make([]string, len(jobs))
					for i, j := range jobs {
						status := "✅"
						if !j.Enabled {
							status = "❌"
						}
						names[i] = fmt.Sprintf("%s %s  (`%s`)", status, j.Name, j.Schedule.Expr)
					}
					return names
				},
			})
		}
	} else {
		log.Printf("[cron] %s not found, skipping", jobsPath)
	}

	log.Println("Gateway is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
	log.Println("Shutting down.")
}
