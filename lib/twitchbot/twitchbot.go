package twitchbot

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"regexp"
	"time"

	"dchristmas.com/twitch-bot/lib/logger"
)


type TwitchBot interface {
	Connect()
	Disconnect()
	HandleChat() error
	WatchChat()
	JoinChannel()
	setToken() error
	Say(msg string) error
	Start()
}

type Bot struct {
	//chat channel
	Channel string

	//Holds our tcp connection to twitch
	conn net.Conn

	//Our connection OAuth token
	Token string

	Keys *Keys
	//https://dev.twitch.tv/docs/irc#irc-command-and-message-limits
	MsgRate time.Duration
	//the bots name
	Name string

	//irc port
	Port string
	//path to oauth creds
	PrivatePath string
	//IRC Domain
	Server string
}

type Keys struct {
	Id     string `json:"client_id_bot"`
	Secret string `json:"client_secret_bot"`
	OAuth string `json:"oauth"`
}

func (bot *Bot) setToken() error {
	f, err := os.Open(".keys.json")
	if err != nil {
		logger.Log("FATAL: filed to open keys files")
		return err
	}

	keys := &Keys{}

	defer f.Close()
	dec := json.NewDecoder(f)
	dec.Decode(keys)

	bot.Token = keys.OAuth

	return nil
}

func (bot *Bot) Start() {
	err := bot.setToken()
	if err != nil {
		panic(err)
	}

	bot.Connect()

	go bot.JoinChannel()
	go bot.WatchChat()
}

func (bot *Bot) Connect() {
	var err error
	logger.Log(fmt.Sprintf("attempting to contect to %s on port %s", bot.Server, bot.Port))

	conf := &tls.Config{}
	dialer := &net.Dialer{}
	bot.conn, err = tls.DialWithDialer(dialer, "tcp", bot.Server+":"+bot.Port, conf)
	if err != nil {
		//TODO: retry should be incremental - https://dev.twitch.tv/docs/irc/guide
		logger.Log(err.Error())
		logger.Log("Connection failed, retrying")

		bot.Connect()
		return
	}
	logger.Log("Connected to " + bot.Server)
}

func (bot *Bot) Disconnect() {
	bot.conn.Close()
	logger.Log("Closed connection from " + bot.Server)
}

func (bot *Bot) JoinChannel() {
	logger.Log("joining #" + bot.Channel)
	bot.conn.Write([]byte("PASS " + bot.Token + "\r\n"))
	bot.conn.Write([]byte("NICK " + bot.Name + "\r\n"))
	bot.conn.Write([]byte("JOIN #" + bot.Channel + "\r\n"))


	logger.Log(fmt.Sprintf("joined #%s as @%s", bot.Channel, bot.Name))
}

func (bot *Bot) Say(msg string) error {
	logger.Log("attempting to say")
	if msg == "" {
		return errors.New("must provide a message to Say")
	}

	_, err := bot.conn.Write([]byte(fmt.Sprintf("PRIVMSG #%s :%s\r\n", bot.Channel, msg)))
	if err != nil {
		return err
	}
	logger.Log(bot.Name + ": " + msg)  
	return nil
}

// Regex for parsing PRIVMSG strings.
//
// First matched group is the user's name and the second matched group is the content of the
// user's message.
var msgRegex *regexp.Regexp = regexp.MustCompile(`^:(\w+)!\w+@\w+\.tmi\.twitch\.tv (PRIVMSG) #\w+(?: :(.*))?$`)

// Regex for parsing user commands, from already parsed PRIVMSG strings.
//
// First matched group is the command name and the second matched group is the argument for the
// command.
var cmdRegex *regexp.Regexp = regexp.MustCompile(`^!(\w+)\s?(\w+)?`)

func (bot *Bot) WatchChat() {
	logger.Log("watching " + bot.Channel)
	for {
		err := bot.HandleChat()
		if err != nil {
			logger.Log("FATAL: Handle Chat Error") 
			logger.Log(err.Error())
		}
	}
}

func (bot *Bot) HandleChat() error {

	tp := textproto.NewReader(bufio.NewReader(bot.conn))

	for {
		line, err := tp.ReadLine()
		if err != nil {
			bot.Disconnect()
			return errors.New("channel read failed")
		}

		logger.Log(line)

		if line == "PING: :time.twitch.tv" {
			bot.conn.Write([]byte("PONG :tmi.twitch.tv\r\n"))
			logger.Log("PONG")
			continue
		} else {
			msgParts := msgRegex.FindStringSubmatch(line)
			if msgParts != nil {
				username := msgParts[1]
				msgType := msgParts[2]

				switch msgType {
				case "PRIVMSG":
					msg := msgParts[3]
					logger.Log(username + ": " + msg) 

					cmdMatches := cmdRegex.FindStringSubmatch(msg)
					if cmdMatches != nil {
						cmd := cmdMatches[1]
						//arg := cmdMatches[2]

						if username == bot.Channel {
							switch cmd {
							case "tbdown":
								logger.Log("shutdown command recieved")
								bot.Disconnect()
								return nil
							default:
								// nadda
							}
						}
					}
				default:
					// nadda
				}
			}
		}
		time.Sleep(bot.MsgRate)
	}
}
