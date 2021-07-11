package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
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

	req, err := http.NewRequest("POST", fmt.Sprintf("https://id.twitch.tv/oauth2/token?client_id=%s&client_secret=%s&grant_type=client_credentials", keys.Id, keys.Secret), strings.NewReader(""))
	if err != nil {
		panic(err)
	}

	var client http.Client

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	io.Copy(os.Stdout, res.Body)
}
