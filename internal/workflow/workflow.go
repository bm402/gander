package workflow

import (
	"github.com/bm402/gander/internal/explore"
	"github.com/bm402/gander/internal/logger"
	"github.com/bm402/gander/internal/retrieval"
)

func DownloadRepoLogs(owner, repo string, threads int) {
	logger.Print(owner, repo, "download-repo-logs", "Creating GitHub client")
	gh := retrieval.CreateGitHubClient()

	logger.Print(owner, repo, "get-run-ids", "Getting run ids")
	runIds := retrieval.GetAllRunIdsForRepo(gh, owner, repo, threads)
	logger.Print(owner, repo, "get-run-ids", "Found", len(runIds), "run ids")

	logger.Print(owner, repo, "download-logs", "Downloading log files")
	downloads := retrieval.DownloadLogsFromRunIds(gh, owner, repo, runIds, threads)
	logger.Print(owner, repo, "download-logs", "Found", downloads, "log files")
}

func SearchLogs(owner, repo, wordlistVariables, wordlistKeywords string, threads int) {
	if len(wordlistVariables) > 0 {
		logger.Print(owner, repo, "search-logs", "Searching logs for variable assignments")
		matches := explore.SearchLogsForVariableAssignments(owner, repo, wordlistVariables, threads)
		logger.Print(owner, repo, "search-logs", "A total of", matches, "distinct variable assignments found")
	} else {
		logger.Print(owner, repo, "search-logs", "No variable names wordlist provided")
	}

	if len(wordlistKeywords) > 0 {
		logger.Print(owner, repo, "search-logs", "Searching logs for keywords")
		matches := explore.SearchLogsForKeywords(owner, repo, wordlistKeywords, threads)
		logger.Print(owner, repo, "search-logs", "A total of", matches, "distinct keywords found")
	} else {
		logger.Print(owner, repo, "search-logs", "No keywords wordlist provided")
	}
}
