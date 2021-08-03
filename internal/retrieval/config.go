package retrieval

import (
	"context"
	"os"

	"github.com/google/go-github/v37/github"
	"golang.org/x/oauth2"
)

func CreateGitHubClient() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: os.Getenv("GH_TOKEN"),
		},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}
