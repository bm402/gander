package workflow

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/bm402/gander/internal/explore"
	"github.com/bm402/gander/internal/githubconfig"
	"github.com/bm402/gander/internal/logger"
	"github.com/bm402/gander/internal/retrieval"
	"github.com/google/go-github/v37/github"
)

type Opts struct {
	Organisation      *string
	Owner             *string
	Repo              *string
	WordlistVariables *string
	WordlistKeywords  *string
	ThreadsDownload   *int
	ThreadsSearch     *int
	IsDownload        *bool
	IsSearch          *bool
	IsOrgRepos        *bool
	IsOrgMembersRepos *bool
}

func Run(opts Opts) {
	if !*opts.IsDownload && !*opts.IsSearch {
		*opts.IsDownload = true
		*opts.IsSearch = true
	}

	var gh *github.Client
	if *opts.IsDownload {
		logger.Print("gander", "", "run", "Creating GitHub client")
		gh = githubconfig.CreateGitHubClient()
	}

	if *opts.Organisation != "" {
		if !*opts.IsOrgRepos && !*opts.IsOrgMembersRepos {
			*opts.IsOrgRepos = true
			*opts.IsOrgMembersRepos = true
		}
		scanOrganisation(gh, opts)
	} else if *opts.Owner != "" && *opts.Repo != "" {
		scanRepoLogs(gh, opts)
	} else {
		logger.Fatal("Incorrect combination of flags used. Either give an -org for a full organisation scan,",
			"or both -owner and -repo for a single repository scan")
	}
}

func scanOrganisation(gh *github.Client, opts Opts) {
	if *opts.IsOrgRepos {
		scanOrganisationRepoLogs(gh, opts)
	}
	if *opts.IsOrgMembersRepos {
		scanOrganisationMembersRepoLogs(gh, opts)
	}
}

func scanOrganisationRepoLogs(gh *github.Client, opts Opts) {
	logger.Print(*opts.Organisation, "", "scan-org-repo-logs", "Getting organisation repos")
	repos := retrieval.GetOrganisationRepos(gh, *opts.Organisation)
	logger.Print(*opts.Organisation, "", "scan-org-repo-logs", "Found", len(repos), "organisation repos")

	for idx, repo := range repos {
		*opts.Owner = *opts.Organisation
		*opts.Repo = repo
		logger.Print(*opts.Owner, *opts.Repo, "scan-org-repo-logs", "Scanning", *opts.Owner+"/"+*opts.Repo,
			fmt.Sprint("(", idx+1, "/", len(repos)), "repos in org)")
		scanRepoLogs(gh, opts)
	}
}

func scanOrganisationMembersRepoLogs(gh *github.Client, opts Opts) {
	logger.Print(*opts.Organisation, "", "scan-org-members-repo-logs", "Getting organisation members")
	members := retrieval.GetOrganisationMembers(gh, *opts.Organisation)
	logger.Print(*opts.Organisation, "", "scan-org-members-repo-logs", "Found", len(members), "members")

	logger.Print(*opts.Organisation, "", "scan-org-members-repo-logs", "Getting organisation members repos")
	repos := retrieval.GetUsersRepos(gh, *opts.Organisation, members, *opts.ThreadsDownload)
	logger.Print(*opts.Organisation, "", "scan-org-members-repo-logs", "Found", len(repos), "repos")

	for idx, repo := range repos {
		parts := strings.Split(repo, "/")
		*opts.Owner = parts[0]
		*opts.Repo = parts[1]
		logger.Print(*opts.Owner, *opts.Repo, "scan-org-members-repo-logs", "Scanning", *opts.Owner+"/"+*opts.Repo,
			fmt.Sprint("(", idx+1, "/", len(repos)), "members repos in org)")
		scanRepoLogs(gh, opts)
	}
}

func scanRepoLogs(gh *github.Client, opts Opts) {
	if *opts.IsDownload {
		downloadRepoLogs(gh, opts)
	}
	if *opts.IsSearch {
		searchRepoLogs(opts)
	}
}

func downloadRepoLogs(gh *github.Client, opts Opts) {
	logger.Print(*opts.Owner, *opts.Repo, "download-logs", "Getting run ids")
	runIds := retrieval.GetAllRunIdsForRepo(gh, *opts.Owner, *opts.Repo, *opts.ThreadsDownload)
	logger.Print(*opts.Owner, *opts.Repo, "download-logs", "Found", len(runIds), "run ids")
	if len(runIds) < 1 {
		logger.Print(*opts.Owner, *opts.Repo, "download-logs", "No logs found, skipping download")
		return
	}

	logger.Print(*opts.Owner, *opts.Repo, "download-logs", "Downloading log files")
	downloads := retrieval.DownloadLogsFromRunIds(gh, *opts.Owner, *opts.Repo, runIds, *opts.ThreadsDownload)
	logger.Print(*opts.Owner, *opts.Repo, "download-logs", "Found", downloads, "log files")
}

func searchRepoLogs(opts Opts) {
	err := exec.Command("ls", *opts.Owner+"/"+*opts.Repo).Run()
	if err != nil {
		logger.Print(*opts.Owner, *opts.Repo, "search-logs", "No logs found, skipping search")
		return
	}

	if len(*opts.WordlistVariables) > 0 {
		logger.Print(*opts.Owner, *opts.Repo, "search-logs", "Searching logs for variable assignments")
		matches := explore.SearchLogsForVariableAssignments(*opts.Owner, *opts.Repo, *opts.WordlistVariables, *opts.ThreadsSearch)
		logger.Print(*opts.Owner, *opts.Repo, "search-logs", "Finished search,", matches, "variable assignments found")
	} else {
		logger.Print(*opts.Owner, *opts.Repo, "search-logs", "No variable names wordlist provided")
	}

	if len(*opts.WordlistKeywords) > 0 {
		logger.Print(*opts.Owner, *opts.Repo, "search-logs", "Searching logs for keywords")
		matches := explore.SearchLogsForKeywords(*opts.Owner, *opts.Repo, *opts.WordlistKeywords, *opts.ThreadsSearch)
		logger.Print(*opts.Owner, *opts.Repo, "search-logs", "Finished search", matches, "keywords found")
	} else {
		logger.Print(*opts.Owner, *opts.Repo, "search-logs", "No keywords wordlist provided")
	}
}
