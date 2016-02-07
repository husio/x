package ghub

import (
	"net/http"

	"github.com/google/go-github/github"
)

func MockClient(c *http.Client) GithubClient {
	return &mockClient{}
}

type mockClient struct {
}

func (c *mockClient) CreateHook(login, repo string, hook *github.Hook) (*github.Hook, error) {
	return nil, nil
}

func (c *mockClient) EditIssue(login, repo string, num int, opts *github.IssueRequest) (*github.Issue, error) {
	return nil, nil
}

func (c *mockClient) ListRepositories(login string, opts *github.RepositoryListOptions) ([]github.Repository, error) {
	return nil, nil
}
