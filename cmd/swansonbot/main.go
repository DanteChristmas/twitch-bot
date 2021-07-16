package main

import (
	"encoding/json"
	"os"

	"dchristmas.com/twitch-bot/pkg/twitchbot"
)

type AppConfig struct {
	Channel string `json:"channel"`
	Name    string `json:"name"`
	Port    string `json:"port"`
	Server  string `json:"server"`
}

func getConfig() *AppConfig {
	f, err := os.Open("config.json")
	if err != nil {
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
		Name:    config.Name,
		Port:    config.Port,
		Server:  config.Server,
	}

	bot.Start()
}
