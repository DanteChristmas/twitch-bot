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
	"time"

	"dchristmas.com/twitch-bot/pkg/logger"
	"dchristmas.com/twitch-bot/pkg/ratelimiter"
	"dchristmas.com/twitch-bot/pkg/swanson"
)

type TwitchBot interface {
	Connect() bool
	Reconnect()
	Disconnect()
	HandleChat() error
	WatchChat()
	JoinChannel()
	SetToken() error
	Say(msg string) error
	Whisper(msg string, user string) error
	Start()
	setLimiters()
	ChatSwanson()
	WhisperSwanson(name string)
}

type Bot struct {
	Channel string
	//Holds our tcp connection to twitch
	conn        net.Conn
	Token       string
	Keys        *Keys
	Name        string
	Port        string
	PrivatePath string
	Server      string

	ChannelLimiter *ratelimiter.Limiter
	WhisperLimiter *ratelimiter.Limiter
}

//Bot Connection Retry config
const (
	CONN_MAX_ATTEMPTS   int = 10
	CONN_RETRY_INTERVAL int = 2
)

type Keys struct {
	Id     string `json:"client_id_bot"`
	Secret string `json:"client_secret_bot"`
	OAuth  string `json:"oauth"error `
}

//Bot Commands
const (
	SWANSON_CMD  string = "!swanson"
	SHUTDOWN_CMD string = "!shutdown"
)

func (bot *Bot) SetToken() error {
	f, err := os.Open(".keys.json")
	if err != nil {
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
	err := bot.SetToken()
	if err != nil {
		panic(err)
	}
	if bot.Reconnect() {
		bot.setLimiters()
		bot.JoinChannel()
	}
}

func (bot *Bot) setLimiters() {
	bot.ChannelLimiter = &ratelimiter.Limiter{}
	bot.WhisperLimiter = &ratelimiter.Limiter{}
	bot.ChannelLimiter.Start(time.Duration(1.5*float64(time.Second.Nanoseconds())), 20)
	bot.WhisperLimiter.Start(time.Duration(time.Second.Nanoseconds()), 3)
}

func (bot *Bot) Reconnect() bool {
	for i := 0; i < CONN_MAX_ATTEMPTS; i++ {
		time.Sleep(time.Duration(i * CONN_RETRY_INTERVAL * int(time.Second)))
		if bot.Connect() {
			return true
		}
		i++
	}
	logger.Log("failed to connect to Twitch IRC, shutting down")
	return false
}

func (bot *Bot) Connect() bool {
	var err error
	logger.Log(fmt.Sprintf("attempting to connect to %s on port %s", bot.Server, bot.Port))

	conf := &tls.Config{}
	dialer := &net.Dialer{Timeout: 1 * time.Second}
	bot.conn, err = tls.DialWithDialer(dialer, "tcp", bot.Server+":"+bot.Port, conf)
	if err != nil {
		logger.Log(err.Error())
		logger.Log("Connection failed, retrying")

		return false
	}

	logger.Log("Connected to " + bot.Server)
	return true
}

func (bot *Bot) Disconnect() {
	bot.conn.Close()
	logger.Log("Closed connection from " + bot.Server)
}

func (bot *Bot) JoinChannel() {
	logger.Log("joining d" + bot.Channel)
	bot.conn.Write([]byte("PASS " + bot.Token + "\r\n"))
	bot.conn.Write([]byte("NICK " + bot.Name + "\r\n"))
	bot.conn.Write([]byte("CAP REQ :twitch.tv/commands\r\n"))
	bot.conn.Write([]byte("JOIN #" + bot.Channel + "\r\n"))

	defer bot.WatchChat()
	logger.Log(fmt.Sprintf("joined %s as #%s", bot.Channel, bot.Name))
}

func (bot *Bot) Say(msg string) error {
	bot.ChannelLimiter.GetToken()

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

// Whispering requires your oauth token to be a verified bot by twitter
func (bot *Bot) Whisper(username string, msg string) error {
	bot.WhisperLimiter.GetToken()

	if msg == "" || username == "" {
		return errors.New("must provide both a message and user to Whisper")
	}

	_, err := bot.conn.Write([]byte(fmt.Sprintf("PRIVMSG #%s :/w %s %s", bot.Channel, username, msg)))
	if err != nil {
		return err
	}
	logger.Log(fmt.Sprintf("#%s: @%s %s", bot.Name, username, msg))
	return nil
}

func (bot *Bot) HandlePing() error {
	logger.Log("#Twitch: PING")
	_, err := bot.conn.Write([]byte("PONG :tmi.twitch.tv\r\n"))
	if err != nil {
		logger.Log("Ping Error")
		logger.Log(err.Error())
	}

	logger.Log(fmt.Sprintf("#%s: PONG", bot.Name))
	return nil
}

func (bot *Bot) WatchChat() {
	logger.Log("watching " + bot.Channel)

	err := bot.HandleChat()
	if err != nil {
		logger.Log("FATAL: Handle Chat Error")
		logger.Log(err.Error())
	}
}

func (bot *Bot) ChatSwanson() {
	quote, err := swanson.GetQuote()
	if err != nil {
		logger.Log(err.Error())
	}

	bot.Say(quote)
}

func (bot *Bot) WhisperSwanson(name string) {
	quote, err := swanson.GetQuote()
	if err != nil {
		logger.Log(err.Error())
	}

	bot.Whisper(name, quote)
}

func (bot *Bot) HandleChat() error {

	tp := textproto.NewReader(bufio.NewReader(bot.conn))

	for {
		line, err := tp.ReadLine()
		if err != nil {
			bot.Disconnect()
			return errors.New("channel read failed")
		}

		msg, err := ParseMessage(line)
		if err != nil {
			return err
		}

		switch msg.Type {
		case PRIVMSG:
			logger.Log(fmt.Sprintf("#%s: %s", msg.Name, msg.Payload))

		case WHISPER:
			logger.Log(fmt.Sprintf("#%s (whisper): %s", msg.Name, msg.Payload))

		case PING:
			go bot.HandlePing()

		case NOTICE:
			logger.Log("#Twitch (Notice): " + msg.Payload)

		case CHATCOMMAND:
			logger.Log(fmt.Sprintf("#%s: %s", msg.Name, msg.Payload))
			switch msg.Payload {
			case SWANSON_CMD:
				go bot.ChatSwanson()

			case SHUTDOWN_CMD:
				logger.Log("Shutdown Command recieved, signing off")
				bot.Disconnect()
				return nil

			default:
				logger.Log(fmt.Sprintf("#%s: %s", msg.Name, msg.Payload))
			}

		case WHISPERCOMMAND:
			logger.Log(fmt.Sprintf("#%s (whisper): %s", msg.Name, msg.Payload))
			switch msg.Payload {
			case SWANSON_CMD:
				go bot.WhisperSwanson(msg.Name)

			case SHUTDOWN_CMD:
				logger.Log("Shutdown Command recieved, signing off")
				bot.Disconnect()
				return nil

			default:
				logger.Log(fmt.Sprintf("#%s (whisper): %s", msg.Name, msg.Payload))
			}
		default:
			logger.Log("Unknown message format received")
			logger.Log(msg.Payload)
		}
	}
}
