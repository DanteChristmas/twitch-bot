package logger

import (
	"fmt"
	"time"
)

const TimeFormat = "Mon Jan 2 15:04:05 MST"

func timeStamp() string {
	return time.Now().Format(TimeFormat)
}

func Log(msg string) {
	fmt.Printf("|%s| %s\n", timeStamp(), msg)
}
