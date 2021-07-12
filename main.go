package main

import (
	"encoding/json"
	"os"
	"time"

	"dchristmas.com/twitch-bot/lib/logger"
	"dchristmas.com/twitch-bot/lib/twitchbot"
)

type AppConfig struct {
	Channel string `json:"channel"`
	Name     string `json:"name"`
	Port string `json:"port"`
	Server string `json:"server"`
}

func getConfig() *AppConfig {
	f, err := os.Open("config.json")
	if err != nil {
		logger.Log("FATAL: failed to open config file")
		logger.Log(err.Error())
		panic(err)
	}

	config := &AppConfig{}

	defer f.Close()
	dec := json.NewDecoder(f)
	dec.Decode(config)

	return config
}

func main() {
	config := getConfig()

	bot := twitchbot.Bot{
		Channel: config.Channel,
		MsgRate: time.Duration(30) * time.Millisecond,
		Name: config.Name,
		Port: config.Port,
		Server: config.Server,
	}

	bot.Start()
}
