package retrieval

import (
	"context"
	"log"
	"math"
	"sync"

	"github.com/google/go-github/v37/github"
)

var PAGE_SIZE = 100

func GetAllLogUrlsForRepo(owner string, repo string) []string {
	gh := createGitHubClient()

	// get first page of workflow runs
	workflowRunsFirstPage := getWorkflowRunsByPage(gh, owner, repo, 1)
	logUrlsFirstPage := getLogUrlsFromWorkflowRuns(workflowRunsFirstPage)

	// calculate totals
	totalWorkflowRuns := *workflowRunsFirstPage.TotalCount
	totalPages := int(math.Ceil(float64(totalWorkflowRuns) / float64(PAGE_SIZE)))

	// create page urls array
	logUrlsByPage := make([][]string, totalPages)
	logUrlsByPage[0] = logUrlsFirstPage

	// get remaining pages of workflow runs
	wg := sync.WaitGroup{}
	for page := 2; page <= totalPages; page++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			logUrlsByPage[page-1] = getLogUrlsByPage(gh, owner, repo, page)
		}(page)
	}
	wg.Wait()

	// combine log url page arrays
	logUrls := []string{}
	for _, logUrlsForPage := range logUrlsByPage {
		logUrls = append(logUrls, logUrlsForPage...)
	}

	return logUrls
}

func getLogUrlsByPage(gh *github.Client, owner string, repo string, page int) []string {
	workflowRuns := getWorkflowRunsByPage(gh, owner, repo, page)
	return getLogUrlsFromWorkflowRuns(workflowRuns)
}

func getWorkflowRunsByPage(gh *github.Client, owner string, repo string, page int) *github.WorkflowRuns {
	workflowRuns, _, err := gh.Actions.ListRepositoryWorkflowRuns(context.TODO(), owner, repo, &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: PAGE_SIZE,
		},
	})

	if err != nil {
		log.Fatal("Could not retrieve workflow runs: ", err.Error())
	}

	return workflowRuns
}

func getLogUrlsFromWorkflowRuns(workflowRuns *github.WorkflowRuns) []string {
	logUrls := []string{}
	for _, workflowRun := range workflowRuns.WorkflowRuns {
		logUrls = append(logUrls, *workflowRun.LogsURL)
	}
	return logUrls
}
