package main

import (
	"time"

	twitchbot "dchristmas.com/twitch-bot/lib"
)

func main() {
	bot := twitchbot.Bot{
		Channel: "dantechristmas_bot",
		MsgRate: time.Duration(20/30) * time.Millisecond,
		Name: "dantechristmas_bot",
		//Port: "6667",
		Port: "6697",
		Server: "irc.chat.twitch.tv",
	}

	bot.Start()
}
