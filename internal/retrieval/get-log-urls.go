package retrieval

import (
	"context"
	"io/ioutil"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/bm402/gander/internal/logger"
	"github.com/google/go-github/v37/github"
)

var PAGE_SIZE = 100

func GetAllRunIdsForRepo(gh *github.Client, owner, repo string, threads int) []int64 {
	// get first page of workflow runs
	workflowRunsFirstPage := getWorkflowRunsByPage(gh, owner, repo, 1, 0)
	if *workflowRunsFirstPage.TotalCount == 0 {
		return []int64{}
	}
	runIdsFirstPage := getRunIdsFromWorkflowRuns(workflowRunsFirstPage)

	// calculate totals
	totalWorkflowRuns := *workflowRunsFirstPage.TotalCount
	totalPages := int(math.Ceil(float64(totalWorkflowRuns) / float64(PAGE_SIZE)))

	// create page ids array
	runIdsByPage := make([][]int64, totalPages)
	runIdsByPage[0] = runIdsFirstPage

	// get remaining pages of workflow runs
	wg := sync.WaitGroup{}
	pages := make(chan int, totalPages-1)

	// create worker threads
	for i := 0; i < threads; i++ {
		go func(pages <-chan int, thread int) {
			for page := range pages {
				runIdsByPage[page-1] = getRunIdsByPage(gh, owner, repo, page, thread)
				wg.Done()
			}
		}(pages, i)
	}

	// add remaining pages to channel to trigger workers
	for j := 2; j <= totalPages; j++ {
		wg.Add(1)
		pages <- j
	}

	// close channel and wait for threads to finish
	close(pages)
	wg.Wait()

	// combine run id page arrays
	runIds := []int64{}
	for _, runIdsForPage := range runIdsByPage {
		runIds = append(runIds, runIdsForPage...)
	}

	return runIds
}

func getRunIdsByPage(gh *github.Client, owner, repo string, page, thread int) []int64 {
	workflowRuns := getWorkflowRunsByPage(gh, owner, repo, page, thread)
	return getRunIdsFromWorkflowRuns(workflowRuns)
}

func getWorkflowRunsByPage(gh *github.Client, owner, repo string, page, thread int) *github.WorkflowRuns {
	workflowRuns, resp, err := gh.Actions.ListRepositoryWorkflowRuns(context.TODO(), owner, repo, &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: PAGE_SIZE,
		},
	})
	defer resp.Body.Close()

	for err != nil {
		respBodyBytes, serr := ioutil.ReadAll(resp.Body)
		if serr != nil {
			respBodyBytes = []byte{}
		}
		// on rate limit, wait and retry
		if _, ok := err.(*github.RateLimitError); ok {
			rateReset := resp.Rate.Reset.Time.Add(time.Minute)
			if thread == 0 {
				logger.Print(owner, repo, "get-run-ids", "Rate limit hit, waiting for reset at", rateReset.String())
			}
			time.Sleep(time.Until(rateReset))
			resp.Body.Close()
			workflowRuns, resp, err = gh.Actions.ListRepositoryWorkflowRuns(context.TODO(), owner, repo, &github.ListWorkflowRunsOptions{
				ListOptions: github.ListOptions{
					Page:    page,
					PerPage: PAGE_SIZE,
				},
			})
		} else if resp.StatusCode == 403 && strings.Contains(string(respBodyBytes), "secondary rate limit") {
			rateReset := time.Now().Add(5 * time.Minute)
			if thread == 0 {
				logger.Print(owner, repo, "get-run-ids", "Secondary rate limit hit, waiting for reset at", rateReset.String())
			}
			time.Sleep(time.Until(rateReset))
			resp.Body.Close()
			workflowRuns, resp, err = gh.Actions.ListRepositoryWorkflowRuns(context.TODO(), owner, repo, &github.ListWorkflowRunsOptions{
				ListOptions: github.ListOptions{
					Page:    page,
					PerPage: PAGE_SIZE,
				},
			})
		} else {
			logger.Print(owner, repo, "get-run-ids", "Could not retrieve page", page, "workflow runs:", err.Error())
			workflowRuns, err = &github.WorkflowRuns{}, nil
		}
	}

	return workflowRuns
}

func getRunIdsFromWorkflowRuns(workflowRuns *github.WorkflowRuns) []int64 {
	runIds := []int64{}
	for _, workflowRun := range workflowRuns.WorkflowRuns {
		runIds = append(runIds, *workflowRun.ID)
	}
	return runIds
}
