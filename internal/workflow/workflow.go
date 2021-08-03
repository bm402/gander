package workflow

import (
	"github.com/bm402/gander/internal/logger"
	"github.com/bm402/gander/internal/retrieval"
)

func GanderRepoOnly(owner, repo string, threads int) {
	logger.Print(owner, repo, "gander-repo-only", "Creating GitHub client")
	gh := retrieval.CreateGitHubClient()

	logger.Print(owner, repo, "get-run-ids", "Getting run ids")
	runIds := retrieval.GetAllRunIdsForRepo(gh, owner, repo, threads)
	logger.Print(owner, repo, "get-run-ids", "Found", len(runIds), "run ids")

	logger.Print(owner, repo, "download-logs", "Downloading log files")
	downloads := retrieval.DownloadLogsFromRunIds(gh, owner, repo, runIds, threads)
	logger.Print(owner, repo, "download-logs", "Found", downloads, "log files")
}
