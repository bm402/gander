package logger

import (
	"fmt"
	"os"
)

func Print(owner, repo, operation string, messages ...interface{}) {
	entry := "[" + owner + "]"
	if len(repo) > 0 {
		entry += "[" + repo + "]"
	}
	entry += "[" + operation + "]"
	for _, message := range messages {
		entry += " " + fmt.Sprint(message)
	}
	fmt.Println(entry)
}

func Fatal(messages ...interface{}) {
	entry := "[fatal]"
	for _, message := range messages {
		entry += " " + fmt.Sprint(message)
	}
	fmt.Println(entry)
	os.Exit(1)
}
