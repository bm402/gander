package retrieval

import (
	"context"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/bm402/gander/internal/logger"
	"github.com/google/go-github/v37/github"
)

func GetOrganisationRepos(gh *github.Client, organisation string) []string {
	repos := []string{}
	page := 1
	isFinished := false

	for !isFinished {
		reposByPage := getOrganisationReposByPage(gh, organisation, page)
		if len(reposByPage) > 0 {
			repos = append(repos, reposByPage...)
			page++
		} else {
			isFinished = true
		}
	}

	return repos
}

func getOrganisationReposByPage(gh *github.Client, organisation string, page int) []string {
	reposData, resp, err := gh.Repositories.ListByOrg(context.TODO(), organisation, &github.RepositoryListByOrgOptions{
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
			logger.Print(organisation, "", "get-org-repos", "Rate limit hit, waiting for reset at", rateReset.String())
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
			logger.Print(organisation, "", "get-org-repos", "Secondary rate limit hit, waiting for reset at", rateReset.String())
			time.Sleep(time.Until(rateReset))
			resp.Body.Close()
			reposData, resp, err = gh.Repositories.ListByOrg(context.TODO(), organisation, &github.RepositoryListByOrgOptions{
				ListOptions: github.ListOptions{
					Page:    page,
					PerPage: PAGE_SIZE,
				},
			})
		} else {
			logger.Print(organisation, "", "get-org-repos", "Could not retrieve page", page, "organisation repos:", err.Error())
			reposData, err = []*github.Repository{}, nil
		}
	}

	// extract repo names
	repos := []string{}
	for _, repoData := range reposData {
		repos = append(repos, *repoData.Name)
	}

	return repos
}
