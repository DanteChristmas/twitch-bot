package twitchbot

import (
	"errors"
	"regexp"
	"strings"
)

type IRCType int
const (
	//Handle messages we haven't accounted for
	UNKNOWN IRCType=-1
	PRIVMSG IRCType=0
	PING IRCType=1
	NOTICE IRCType=2
	WHISPER IRCType=3
	
	CHATCOMMAND IRCType=4
	WHISPERCOMMAND IRCType=5
)


const (
	PRIVMSG_STR string = "PRIVMSG"
	NOTICE_STR string = "NOTICE"
	WHISPER_STR string = "WHISPER"
)

const PING_MESSAGE = "PING :tmi.twitch.tv"

type Message struct {
	Payload string
	Name string
	Type IRCType
}

// Regex for parsing standard IRC message strings.
//var msgRegex *regexp.Regexp = regexp.MustCompile(`^:(\w+)!\w+@\w+\.tmi\.twitch\.tv ([A-Z]*) #\w+(?: :(.*))?$`)
var msgRegex *regexp.Regexp = regexp.MustCompile(`^:(\w+)!\w+@\w+\.tmi\.twitch\.tv (PRIVMSG|NOTICE) #\w+(?: :(.*))?$`)
var whisperRegex *regexp.Regexp = regexp.MustCompile(`^:(\w+)!\w+@\w+\.tmi\.twitch\.tv (WHISPER) \w+(?: :(.*))?$`)

// Regex for parsing user commands, from already parsed PRIVMSG strings.
var cmdRegex *regexp.Regexp = regexp.MustCompile(`^!(\w+)\s?(\w+)?`)

func ParseMessage(line string) (*Message, error) {
	if line == PING_MESSAGE {
		return &Message{
			Payload: "",
			Type: PING,
		}, nil
	}

	msgParts := strings.Split(line, " ")
	if len(msgParts) < 2 {
		return nil, errors.New("message parse error")
	}
	msgType := msgParts[1]

	switch msgType {
	case PRIVMSG_STR:
		chatParts := msgRegex.FindStringSubmatch(line)
		if chatParts == nil {
			return nil, errors.New("notice parse error")
		}

		cmdPayload := cmdRegex.FindStringSubmatch(chatParts[3])
		if cmdPayload != nil {
			return &Message{
				Payload: cmdPayload[0],
				Type: CHATCOMMAND,
				Name: chatParts[1],
			}, nil
		}

		return &Message{
			Payload: chatParts[3],
			Name: chatParts[1],
			Type: PRIVMSG,
		}, nil

	case NOTICE_STR:
		noticeParts := msgRegex.FindStringSubmatch(line)
		if noticeParts == nil {
			return nil, errors.New("chat parse error")
		}
		
		return &Message{
			Payload: noticeParts[3],
			Name: "Twitch",
			Type: PRIVMSG,
		}, nil

	case WHISPER_STR:
		chatParts := whisperRegex.FindStringSubmatch(line)
		if chatParts == nil {
			return nil, errors.New("chat parse error")
		}

		cmdPayload := cmdRegex.FindStringSubmatch(chatParts[3])
		if cmdPayload != nil {
			return &Message{
				Payload: cmdPayload[0],
				Type: WHISPERCOMMAND,
				Name: chatParts[1],
			}, nil
		}

		return &Message{
			Payload: chatParts[3],
			Name: chatParts[1],
			Type: WHISPER,
		}, nil

	default: 
		return &Message{
			Payload: line,
			Type: UNKNOWN,
		}, nil
	}
}
