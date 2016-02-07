package ghub

import (
	"net/http"

	"github.com/google/go-github/github"
	"golang.org/x/net/context"
)

func WithClient(ctx context.Context, c ClientCreator) context.Context {
	return context.WithValue(ctx, "github:clientcreator", c)
}

type ClientCreator func(*http.Client) GithubClient

func Client(ctx context.Context, c *http.Client) GithubClient {
	val := ctx.Value("github:clientcreator")
	if val == nil {
		panic("github client creator not present in the context")
	}
	cc := val.(ClientCreator)
	return cc(c)
}

type GithubClient interface {
	CreateHook(login, repo string, hook *github.Hook) (*github.Hook, error)
	EditIssue(login, repo string, num int, opts *github.IssueRequest) (*github.Issue, error)
	ListRepositories(login string, opts *github.RepositoryListOptions) ([]github.Repository, error)
}

func StandardClient(c *http.Client) GithubClient {
	return &stdClient{ghc: github.NewClient(c)}
}

type stdClient struct {
	ghc *github.Client
}

func (c *stdClient) CreateHook(login, repo string, hook *github.Hook) (*github.Hook, error) {
	hook, _, err := c.ghc.Repositories.CreateHook(login, repo, hook)
	return hook, err
}

func (c *stdClient) EditIssue(login, repo string, num int, opts *github.IssueRequest) (*github.Issue, error) {
	issue, _, err := c.ghc.Issues.Edit(login, repo, num, opts)
	return issue, err
}

func (c *stdClient) ListRepositories(login string, opts *github.RepositoryListOptions) ([]github.Repository, error) {
	repos, _, err := c.ghc.Repositories.List(login, opts)
	return repos, err
}
