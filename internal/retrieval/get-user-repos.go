package retrieval

import (
	"context"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bm402/gander/internal/logger"
	"github.com/google/go-github/v37/github"
)

type userWithLocalId struct {
	user    string
	localId int
}

func GetUsersRepos(gh *github.Client, organisation string, users []string, threads int) []string {
	wg := sync.WaitGroup{}
	usernames := make(chan userWithLocalId, len(users))
	reposByUser := make([][]string, len(users))

	// create worker threads
	for i := 0; i < threads; i++ {
		go func(users <-chan userWithLocalId, thread int) {
			for user := range users {
				repos := []string{}
				page := 1
				isFinished := false
				for !isFinished {
					reposByPage := getUserReposByPage(gh, organisation, user.user, page)
					if len(reposByPage) > 0 {
						repos = append(repos, reposByPage...)
						page++
					} else {
						isFinished = true
					}
				}
				reposByUser[user.localId] = repos
				wg.Done()
			}
		}(usernames, i)
	}

	// add users to channel to trigger workers
	for j := 0; j < len(users); j++ {
		wg.Add(1)
		usernames <- userWithLocalId{
			user:    users[j],
			localId: j,
		}
	}

	// close channel and wait for threads to finish
	close(usernames)
	wg.Wait()

	// combine repos to single array
	repos := []string{}
	for _, userRepos := range reposByUser {
		repos = append(repos, userRepos...)
	}

	return repos
}

func getUserReposByPage(gh *github.Client, organisation, user string, page int) []string {
	reposData, resp, err := gh.Repositories.List(context.TODO(), user, &github.RepositoryListOptions{
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
		if resp.StatusCode == 403 && resp.Rate.Remaining == 0 && resp.Rate.Reset.Time.After(time.Now()) {
			rateReset := resp.Rate.Reset.Time.Add(time.Minute)
			logger.Print(organisation, user, "get-user-repos", "Rate limit hit, waiting for reset at", rateReset.String())
			time.Sleep(time.Until(rateReset))
			resp.Body.Close()
			reposData, resp, err = gh.Repositories.ListByOrg(context.TODO(), organisation, &github.RepositoryListByOrgOptions{
				ListOptions: github.ListOptions{
					Page:    page,
					PerPage: PAGE_SIZE,
				},
			})
		} else if resp.StatusCode == 403 && strings.Contains(string(respBodyBytes), "secondary rate limit") {
			var rateReset time.Time
			if retryAfters, ok := resp.Header["Retry-After"]; ok {
				retryAfter, _ := strconv.Atoi(retryAfters[0])
				rateReset = time.Now().Add(time.Duration(retryAfter+5) * time.Second)
			} else {
				rateReset = time.Now().Add(5 * time.Minute)
			}
			logger.Print(organisation, user, "get-user-repos", "Secondary rate limit hit, waiting for reset at", rateReset.String())
			time.Sleep(time.Until(rateReset))
			resp.Body.Close()
			reposData, resp, err = gh.Repositories.ListByOrg(context.TODO(), organisation, &github.RepositoryListByOrgOptions{
				ListOptions: github.ListOptions{
					Page:    page,
					PerPage: PAGE_SIZE,
				},
			})
		} else {
			logger.Print(organisation, user, "get-user-repos", "Could not retrieve page", page, "user repos:", err.Error())
			reposData, err = []*github.Repository{}, nil
		}
	}

	// extract repo names
	repos := []string{}
	for _, repoData := range reposData {
		repos = append(repos, user+"/"+*repoData.Name)
	}

	return repos
}
