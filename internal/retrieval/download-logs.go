package retrieval

import (
	"context"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bm402/gander/internal/logger"
	"github.com/google/go-github/v37/github"
	"github.com/google/uuid"
)

type runIdConfig struct {
	id    int64
	count int
}

func DownloadLogsFromRunIds(gh *github.Client, owner, repo string, runIds []int64, threads int) int {
	wg := sync.WaitGroup{}
	runIdConfigs := make(chan runIdConfig, len(runIds))
	fivePercent := len(runIds) / 20
	successfulDownloads := int64(0)

	// create worker threads
	for i := 0; i < threads; i++ {
		go func(idConfigs <-chan runIdConfig) {
			for idConfig := range idConfigs {
				// status update in 5% increments
				if len(runIds) >= 20 && idConfig.count > 0 && idConfig.count%fivePercent == 0 {
					logger.Print(owner, repo, "download-logs", idConfig.count, "downloads attempted")
				}
				err := getLogsFromRunId(gh, owner, repo, idConfig.id)
				if err == nil {
					atomic.AddInt64(&successfulDownloads, 1)
				}
				wg.Done()
			}
		}(runIdConfigs)
	}

	// add urls to channel to trigger workers
	for j := 0; j < len(runIds); j++ {
		wg.Add(1)
		runIdConfigs <- runIdConfig{
			id:    runIds[j],
			count: j,
		}
	}

	// close channel and wait for threads to finish
	close(runIdConfigs)
	wg.Wait()

	return int(successfulDownloads)
}

func getLogsFromRunId(gh *github.Client, owner, repo string, runId int64) error {
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

	url, err := getLogUrl(gh, owner, repo, runId)
	if err != nil {
		return err
	}
	err = downloadLogArchive(owner, repo, url, foldername.String())
	if err != nil {
		return err
	}
	err = unzipLogArchive(owner, repo, foldername.String())
	if err != nil {
		return err
	}
	deleteDuplicateLogFiles(owner, repo, foldername.String())
	addRunIdToFolder(owner, repo, runId, foldername.String())
	return nil
}

func getLogUrl(gh *github.Client, owner, repo string, runId int64) (string, error) {
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
			return "", err
		}
	}
	return redirectUrl.String(), nil
}

func downloadLogArchive(owner, repo, url, foldername string) error {
	err := exec.Command("wget", "-O", foldername+".zip", url).Run()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not download the log archive:", err.Error())
	}
	return err
}

func unzipLogArchive(owner, repo, foldername string) error {
	err := exec.Command("unzip", "-d", foldername, foldername+".zip").Run()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not unzip the log archive:", err.Error())
		return err
	}

	err = exec.Command("rm", foldername+".zip").Run()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not delete the unzipped log archive:", err.Error())
	}
	return err
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
