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
	threadsDownload := flag.Int("td", 5, "Number of threads for download (be wary of GitHub API rate limits)")
	threadsSearch := flag.Int("ts", 20, "Number of threads for search")
	flag.Parse()

	if !*isDownload && !*isSearch {
		*isDownload = true
		*isSearch = true
	}

	if len(*owner) < 1 || len(*repo) < 1 {
		logger.Fatal("Owner and/or repo flags not set")
	}

	if *isDownload {
		workflow.DownloadRepoLogs(*owner, *repo, *threadsDownload)
	}

	if *isSearch {
		workflow.SearchLogs(*owner, *repo, *wordlistVariables, *wordlistKeywords, *threadsSearch)
	}
}
