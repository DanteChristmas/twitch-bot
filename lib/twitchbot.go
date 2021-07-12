package twitchbot

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"regexp"
	"time"
)

type Keys struct {
	Id     string `json:"client_id_bot"`
	Secret string `json:"client_secret_bot"`
	OAuth string `json:"oauth"`
}

type TokenResponse struct {
	Token     string `json:"access_token"`
	Expires   int    `json:"expires_in"`
	TokenType string `json:"token_type"`
}

func (bot *Bot) ReadCredentials() error {
	f, err := os.Open(".keys.json")
	if err != nil {
		fmt.Printf("|%s| Failed to open keys file", timeStamp())
		return err
	}

	keys := &Keys{}

	defer f.Close()
	dec := json.NewDecoder(f)
	dec.Decode(keys)

//	req, err := http.NewRequest("POST", fmt.Sprintf("https://id.twitch.tv/oauth2/token?client_id=%s&client_secret=%s&grant_type=client_credentials&scopes=chat:read,chat:edit", keys.Id, keys.Secret), strings.NewReader(""))
//	if err != nil {
//		fmt.Printf("|%s| failed to build auth request", timeStamp())
//		return err
//	}
//	req.Header.Add("content-type", "application/json")
//
//	var client http.Client
//
//	res, err := client.Do(req)
//	if err != nil {
//		bot.Disconnect()
//		return errors.New("oauth token fetch failure")
//	}
//	defer res.Body.Close()
//
//	token := &TokenResponse{}
//	json.NewDecoder(res.Body).Decode(token)
//
//	bot.Token = "oauth:" + token.Token
//

	bot.Token = keys.OAuth
	fmt.Println(bot.Token)

	return nil
}

type TwitchBot interface {
	Connect()
	Disconnect()
	HandleChat() error
	JoinChannel()
	ReadCredentials() (string, error)
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

const TimeFormat = "Mon Jan 2 15:04:05 MST"

func timeStamp() string {
	return TimeStamp(TimeFormat)
}

func TimeStamp(format string) string {
	return time.Now().Format(format)
}

func (bot *Bot) Start() {
	err := bot.ReadCredentials()
	if err != nil {
		panic(err)
	}

	for {
		bot.Connect()
		bot.JoinChannel()
		err = bot.HandleChat()
		if err != nil {
			time.Sleep(1000 * time.Millisecond)
			fmt.Printf("|%s| %s\n", timeStamp(), err)
			fmt.Printf("|%s| restaring bot\n", timeStamp())
		} else {
			return
		}
	}
}

func (bot *Bot) Connect() {
	var err error
	fmt.Printf("|%s| Attempting to connect to %s on port %s\n", timeStamp(), bot.Server, bot.Port)

	bot.conn, err = net.Dial("tcp", bot.Server+":"+bot.Port)
	if err != nil {
		//TODO: retry should be incremental - https://dev.twitch.tv/docs/irc/guide
		fmt.Println(err)
		fmt.Printf("|%s| Connection failed, retrying\n", timeStamp())
		bot.Connect()
		return
	}
	fmt.Printf("|%s| Connected to %s\n", timeStamp(), bot.Server)
}

func (bot *Bot) Disconnect() {
	bot.conn.Close()
	fmt.Printf("|%s| Closed connection from %s\n", timeStamp(), bot.Server)
}

func (bot *Bot) JoinChannel() {
	fmt.Printf("|%s| joining #%s\n", timeStamp(), bot.Channel)
	bot.conn.Write([]byte("PASS " + bot.Token + "\r\n"))
	bot.conn.Write([]byte("NICK " + bot.Name + "\r\n"))
	bot.conn.Write([]byte("JOIN #" + bot.Channel + "\r\n"))

	fmt.Printf("|%s| joined #%s as @%s\n", timeStamp(), bot.Channel, bot.Name)
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

func (bot *Bot) HandleChat() error {
	fmt.Printf("|%s| watching %s\n", timeStamp(), bot.Channel)

	tp := textproto.NewReader(bufio.NewReader(bot.conn))

	for {
		line, err := tp.ReadLine()
		if err != nil {
			bot.Disconnect()
			return errors.New("channel read failed")
		}

		fmt.Printf("|%s| %s\n", timeStamp(), line)

		if line == "PING: :time.twitch.tv" {
			bot.conn.Write([]byte("PONG :tmi.twitch.tv\r\n"))
			continue
		} else {
			msgParts := msgRegex.FindStringSubmatch(line)
			if msgParts != nil {
				username := msgParts[1]
				msgType := msgParts[2]

				switch msgType {
				case "PRIVMSG":
					msg := msgParts[3]
					fmt.Printf("|%s| %s: %s\n", timeStamp(), username, msg)

					cmdMatches := cmdRegex.FindStringSubmatch(msg)
					if cmdMatches != nil {
						cmd := cmdMatches[1]
						//arg := cmdMatches[2]

						if username == bot.Channel {
							switch cmd {
							case "tbdown":
								fmt.Printf("|%s| shutdown command recieved", timeStamp())
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
