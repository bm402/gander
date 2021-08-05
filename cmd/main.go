package main

import (
	"flag"

	"github.com/bm402/gander/internal/logger"
	"github.com/bm402/gander/internal/workflow"
)

func main() {
	isDownload := flag.Bool("download", false, "Run the log download from GitHub")
	isSearch := flag.Bool("search", false, "Run the search on existing logs in the current directory")
	owner := flag.String("o", "", "The owner of the repository")
	repo := flag.String("r", "", "The name of the repository")
	wordlistVariables := flag.String("wv", "", "The wordlist of variable names")
	wordlistKeywords := flag.String("wk", "", "The wordlist of keywords")
	threads := flag.Int("t", 10, "Number of threads")
	flag.Parse()

	if !*isDownload && !*isSearch {
		*isDownload = true
		*isSearch = true
	}

	if len(*owner) < 1 || len(*repo) < 1 {
		logger.Fatal("Owner and/or repo flags not set")
	}

	if *isDownload {
		workflow.DownloadRepoLogs(*owner, *repo, *threads)
	}

	if *isSearch {
		workflow.SearchLogs(*owner, *repo, *wordlistVariables, *wordlistKeywords, *threads)
	}
}
