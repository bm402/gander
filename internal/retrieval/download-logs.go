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
		go func(idConfigs <-chan runIdConfig, thread int) {
			for idConfig := range idConfigs {
				// status update in 5% increments
				if len(runIds) >= 20 && idConfig.count > 0 && idConfig.count%fivePercent == 0 {
					logger.Print(owner, repo, "download-logs", idConfig.count, "downloads attempted",
						"("+strconv.Itoa((idConfig.count/fivePercent)*5)+"%)")
				}
				err := getLogsFromRunId(gh, owner, repo, idConfig.id, thread)
				if err == nil {
					atomic.AddInt64(&successfulDownloads, 1)
				}
				wg.Done()
			}
		}(runIdConfigs, i)
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

func getLogsFromRunId(gh *github.Client, owner, repo string, runId int64, thread int) error {
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

	url, err := getLogUrl(gh, owner, repo, runId, thread)
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

func getLogUrl(gh *github.Client, owner, repo string, runId int64, thread int) (string, error) {
	redirectUrl, resp, err := gh.Actions.GetWorkflowRunLogs(context.TODO(), owner, repo, runId, true)
	defer resp.Body.Close()

	for err != nil {
		respBodyBytes, serr := ioutil.ReadAll(resp.Body)
		if serr != nil {
			respBodyBytes = []byte{}
		}
		// on rate limit, wait and retry
		if resp.StatusCode == 403 && resp.Rate.Remaining == 0 && resp.Rate.Reset.Time.After(time.Now()) {
			rateReset := resp.Rate.Reset.Time.Add(time.Minute)
			if thread == 0 {
				logger.Print(owner, repo, "download-logs", "Rate limit hit, waiting for reset at", rateReset.String())
			}
			time.Sleep(time.Until(rateReset))
			resp.Body.Close()
			redirectUrl, resp, err = gh.Actions.GetWorkflowRunLogs(context.TODO(), owner, repo, runId, true)
		} else if resp.StatusCode == 403 && strings.Contains(string(respBodyBytes), "secondary rate limit") {
			rateReset := time.Now().Add(5 * time.Minute)
			if thread == 0 {
				logger.Print(owner, repo, "download-logs", "Secondary rate limit hit, waiting for reset at", rateReset.String())
			}
			time.Sleep(time.Until(rateReset))
			resp.Body.Close()
			redirectUrl, resp, err = gh.Actions.GetWorkflowRunLogs(context.TODO(), owner, repo, runId, true)
		} else {
			// logger.Print(owner, repo, "download-logs", "Could not get redirect url:", err.Error())
			return "", err
		}
	}

	return redirectUrl.String(), nil
}

func downloadLogArchive(owner, repo, url, foldername string) error {
	err := exec.Command("mkdir", "-p", owner).Run()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not create", owner, "directory:", err.Error())
	}
	err = exec.Command("mkdir", "-p", owner+"/"+repo).Run()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not create", owner+"/"+repo, "directory:", err.Error())
	}
	err = exec.Command("wget", "-O", owner+"/"+repo+"/"+foldername+".zip", url).Run()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not download the log archive:", err.Error())
	}
	return err
}

func unzipLogArchive(owner, repo, foldername string) error {
	err := exec.Command("unzip", "-d", owner+"/"+repo+"/"+foldername, owner+"/"+repo+"/"+foldername+".zip").Run()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not unzip the log archive:", err.Error())
		return err
	}

	err = exec.Command("rm", owner+"/"+repo+"/"+foldername+".zip").Run()
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not delete the unzipped log archive:", err.Error())
	}
	return err
}

func deleteDuplicateLogFiles(owner, repo, foldername string) {
	folderOutput, err := exec.Command("find", owner+"/"+repo+"/"+foldername, "-mindepth", "1", "-maxdepth", "1", "-type", "d").Output()
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
	contents := []byte(strconv.FormatInt(runId, 10) + "\n")
	err := ioutil.WriteFile(owner+"/"+repo+"/"+foldername+"/id", contents, 0644)
	if err != nil {
		logger.Print(owner, repo, "download-logs", "Could not write run id to folder:", err.Error())
	}
}
