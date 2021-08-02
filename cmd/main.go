package main

import (
	"flag"
	"log"

	"github.com/bm402/gander/internal/workflow"
)

var threads *int

func main() {
	owner := flag.String("o", "", "The owner of the repository")
	repo := flag.String("r", "", "The name of the repository")
	threads = flag.Int("t", 10, "Number of threads")
	flag.Parse()

	if len(*owner) < 1 || len(*repo) < 1 {
		log.Fatal("Owner and/or repo flags not set")
	}

	workflow.GanderRepoOnly(*owner, *repo)
}
