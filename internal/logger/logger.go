package logger

import (
	"fmt"
	"os"
)

func Print(owner string, repo string, operation string, messages ...interface{}) {
	fmt.Print("[", owner, "][", repo, "][", operation, "]")
	for _, message := range messages {
		fmt.Print(" ", message)
	}
	fmt.Println()
}

func Fatal(messages ...interface{}) {
	fmt.Print("[fatal]")
	for _, message := range messages {
		fmt.Print(" ", message)
	}
	fmt.Println()
	os.Exit(1)
}
