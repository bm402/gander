package retrieval

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/bm402/gander/internal/logger"
	"github.com/google/go-github/v37/github"
)

var PAGE_SIZE = 100

func GetAllLogUrlsForRepo(gh *github.Client, owner string, repo string, threads int) []string {
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
	pages := make(chan int, totalPages-1)

	// create worker threads
	for i := 0; i < threads; i++ {
		go func(pages <-chan int) {
			for page := range pages {
				logUrlsByPage[page-1] = getLogUrlsByPage(gh, owner, repo, page)
				wg.Done()
			}
		}(pages)
	}

	// add remaining pages to channel to trigger workers
	for j := 2; j <= totalPages; j++ {
		wg.Add(1)
		pages <- j
	}

	// close channel and wait for threads to finish
	close(pages)
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
	workflowRuns, resp, err := gh.Actions.ListRepositoryWorkflowRuns(context.TODO(), owner, repo, &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: PAGE_SIZE,
		},
	})

	for err != nil {
		// on rate limit, wait and retry
		if _, ok := err.(*github.RateLimitError); ok {
			rateReset := resp.Rate.Reset
			logger.Print(owner, repo, "get-log-urls", "Rate limit hit, waiting for reset at", rateReset.String())
			time.Sleep(time.Until(rateReset.Time))
			workflowRuns, resp, err = gh.Actions.ListRepositoryWorkflowRuns(context.TODO(), owner, repo, &github.ListWorkflowRunsOptions{
				ListOptions: github.ListOptions{
					Page:    page,
					PerPage: PAGE_SIZE,
				},
			})
		} else {
			logger.Print(owner, repo, "get-log-urls", "Could not retrieve workflow runs: ", err.Error())
		}
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
