package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

func main() { 
	// Getting a token
	var keys struct{
		Id string `json:"client_id"`
		Secret string `json:"client_secret"`
	}

	f, err := os.Open(".keys.json")
	if err != nil {
		panic(err)
	}

	defer f.Close()
	dec := json.NewDecoder(f)
	dec.Decode(&keys)
	fmt.Println(keys)

	req, err := http.NewRequest("POST", fmt.Sprintf("https://id.twitch.tv/oauth2/token?client_id=%s&client_secret=%s&grant_type=client_credentials&scopes=chat:read,chat:edit", keys.Id, keys.Secret), strings.NewReader(""))
	if err != nil {
		panic(err)
	}
	req.Header.Add("content-type", "application/json")

	var client http.Client

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	var token struct {
		Token string `json:"access_token"`
		Expires int `json:"expires_in"`
		TokenType string `json:"token_type"`
	}

	json.NewDecoder(res.Body).Decode(&token)

	// Connecting to IRC
	c,err := net.Dial("tcp", "irc.chat.twitch.tv:6697")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	pass_message := fmt.Sprintf("PASS oauth:%s\r\n", token.Token)
	c.Write([]byte(pass_message))
	c.Write([]byte("NICK basic botch\r\n"))
	c.Write([]byte("JOIN dantechristmas"))

	var buf bytes.Buffer
	io.Copy(&buf, c)
	fmt.Println("total size:", buf.Len())
}
