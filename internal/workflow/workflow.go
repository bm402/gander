package workflow

import (
	"log"

	"github.com/bm402/gander/internal/retrieval"
)

func GanderRepoOnly(owner string, repo string) {
	log.Print(retrieval.GetAllLogUrlsForRepo(owner, repo))
}
