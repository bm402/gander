package retrieval

import (
	"context"
	"io/ioutil"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bm402/gander/internal/logger"
	"github.com/google/go-github/v37/github"
	"github.com/google/uuid"
)

func DownloadLogsFromRunIds(gh *github.Client, owner, repo string, runIds []int64, threads int) {
	wg := sync.WaitGroup{}
	ids := make(chan int64, len(runIds))

	// create worker threads
	for i := 0; i < threads; i++ {
		go func(ids <-chan int64) {
			for id := range ids {
				getLogsFromRunId(gh, owner, repo, id)
				wg.Done()
			}
		}(ids)
	}

	// add urls to channel to trigger workers
	for j := 0; j < len(runIds); j++ {
		wg.Add(1)
		ids <- runIds[j]
	}

	// close channel and wait for threads to finish
	close(ids)
	wg.Wait()
}

func getLogsFromRunId(gh *github.Client, owner, repo string, runId int64) {
	foldername, err := uuid.NewRandom()
	retries := 0
	for err != nil {
		if retries >= 10 {
			logger.Fatal(owner, repo, "download-logs", "Could not create random uuid filename, quitting")
		}
		logger.Print(owner, repo, "download-logs", "Could not create random uuid filename, retrying")
		retries++
		foldername, err = uuid.NewRandom()
	}

	url := getLogUrl(gh, owner, repo, runId)
	downloadLogArchive(owner, repo, url, foldername.String())
	unzipLogArchive(owner, repo, foldername.String())
	deleteDuplicateLogFiles(owner, repo, foldername.String())
	addRunIdToFolder(owner, repo, runId, foldername.String())
}

func getLogUrl(gh *github.Client, owner, repo string, runId int64) string {
	redirectUrl, resp, err := gh.Actions.GetWorkflowRunLogs(context.TODO(), owner, repo, runId, true)
	for err != nil {
		// on rate limit, wait and retry
		if _, ok := err.(*github.RateLimitError); ok {
			rateReset := resp.Rate.Reset
			logger.Print(owner, repo, "download-logs", "Rate limit hit, waiting for reset at", rateReset.String())
			time.Sleep(time.Until(rateReset.Time))
			redirectUrl, resp, err = gh.Actions.GetWorkflowRunLogs(context.TODO(), owner, repo, runId, true)
		} else {
			logger.Print(owner, repo, "download-logs", "Could not get redirect url:", err.Error())
			redirectUrl, err = &url.URL{}, nil
		}
	}
	return redirectUrl.String()
}

func downloadLogArchive(owner, repo, url, foldername string) {
	err := exec.Command("wget", "-O", foldername+".zip", url).Run()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not download the log archive:", err.Error())
	}
}

func unzipLogArchive(owner, repo, foldername string) {
	err := exec.Command("unzip", "-d", foldername, foldername+".zip").Run()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not unzip the log archive:", err.Error())
		return
	}

	err = exec.Command("rm", foldername+".zip").Run()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not delete the unzipped log archive:", err.Error())
	}
}

func deleteDuplicateLogFiles(owner, repo, foldername string) {
	folderOutput, err := exec.Command("find", foldername, "-mindepth", "1", "-maxdepth", "1", "-type", "d").Output()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not find the duplicate directories:", err.Error())
		return
	}

	folders := strings.Split(string(folderOutput), "\n")
	for _, folder := range folders {
		err := exec.Command("rm", "-rf", folder).Run()
		if err != nil {
			logger.Print(owner, repo, "download-logs", "Could not delete the duplicate directories:", err.Error())
		}
	}
}

func addRunIdToFolder(owner, repo string, runId int64, foldername string) {
	contents := []byte(strconv.FormatInt(runId, 10))
	err := ioutil.WriteFile(foldername+"/id", contents, 0644)
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not write run id to folder:", err.Error())
	}
}
