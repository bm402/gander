package workflow

import (
	"log"

	"github.com/bm402/gander/internal/retrieval"
)

func GanderRepoOnly(owner string, repo string, threads int) {
	gh := retrieval.CreateGitHubClient()
	logUrls := retrieval.GetAllLogUrlsForRepo(gh, owner, repo, threads)
	log.Print(logUrls)
	retrieval.DownloadLogsFromUrls(gh, logUrls)
}
