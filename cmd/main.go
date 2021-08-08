package main

import (
	"flag"

	"github.com/bm402/gander/internal/workflow"
)

func main() {
	opts := workflow.Opts{
		Organisation:      flag.String("org", "", "The organisation to scan"),
		Owner:             flag.String("owner", "", "The owner of the repository"),
		Repo:              flag.String("repo", "", "The name of the repository"),
		WordlistVariables: flag.String("wv", "", "The wordlist of variable names"),
		WordlistKeywords:  flag.String("wk", "", "The wordlist of keywords"),
		ThreadsDownload:   flag.Int("td", 5, "Number of threads for download (be wary of GitHub API rate limits)"),
		ThreadsSearch:     flag.Int("ts", 20, "Number of threads for search"),
		IsDownload:        flag.Bool("download", false, "Run the log download from GitHub"),
		IsSearch:          flag.Bool("search", false, "Run the search on existing logs in the current directory"),
		IsOrgRepos:        flag.Bool("org-repos", false, "Run for organisation repos"),
		IsOrgMembersRepos: flag.Bool("org-members", false, "Run for organisation members repos"),
	}
	flag.Parse()
	workflow.Run(opts)
}
