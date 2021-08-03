package logger

import (
	"fmt"
	"os"
)

func Print(owner, repo, operation string, messages ...interface{}) {
	entry := "[" + owner + "][" + repo + "][" + operation + "]"
	for _, message := range messages {
		entry += " " + fmt.Sprint(message)
	}
	fmt.Println(entry)
}

func Fatal(messages ...interface{}) {
	fmt.Print("[fatal]")
	for _, message := range messages {
		fmt.Print(" ", message)
	}
	fmt.Println()
	os.Exit(1)
}
