package swanson

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"net/textproto"
)

func GetQuote() (string, error) {
	res, err := http.Get("https://ron-swanson-quotes.herokuapp.com/v2/quotes")
	if err != nil {
		return "", errors.New("swawnson api read error")
	}
	defer res.Body.Close()

	var result []string
	reader := textproto.NewReader(bufio.NewReader(res.Body))
	line, err := reader.ReadLine()
	if err != nil {
		return "", errors.New("swawnson api read error")
	}

	err = json.Unmarshal([]byte(line), &result)
	if err != nil {
		return "", errors.New("swawnson api read error")
	}

	return result[0], nil
}
