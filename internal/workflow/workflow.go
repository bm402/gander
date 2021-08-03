package workflow

import (
	"github.com/bm402/gander/internal/logger"
	"github.com/bm402/gander/internal/retrieval"
)

func GanderRepoOnly(owner string, repo string, threads int) {
	gh := retrieval.CreateGitHubClient()
	logUrls := retrieval.GetAllLogUrlsForRepo(gh, owner, repo, threads)
	logger.Print(owner, repo, "gander-repo-only", logUrls)
	retrieval.DownloadLogsFromUrls(gh, logUrls)
}
