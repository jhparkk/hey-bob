#!/bin/bash
set -e

export PATH=$PATH:/usr/local/go/bin
cd "$(dirname "$0")"

echo "[build] building..."
go build -o gateway-claude-discord .

echo "[deploy] restarting service..."
sudo cp gateway-claude-discord.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl restart gateway-claude-discord

echo "[done] $(systemctl is-active gateway-claude-discord)"
