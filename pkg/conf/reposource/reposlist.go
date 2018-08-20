package reposource

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/sourcegraph/sourcegraph/pkg/api"
	"github.com/sourcegraph/sourcegraph/pkg/conf"
	"github.com/sourcegraph/sourcegraph/schema"
)

var (
	reposListInstance *reposList
	reposListMu       sync.Mutex
)

func init() {
	conf.Watch(func() {
		newReposListInstance := newReposList(conf.Get().ReposList)

		reposListMu.Lock()
		reposListInstance = newReposListInstance
		reposListMu.Unlock()
	})
}

func ReposListCloneURLToRepoURI(cloneURL string) (api.RepoURI, error) {
	reposListMu.Lock()
	r := reposListInstance
	reposListMu.Unlock()

	return r.CloneURLToRepoURI(cloneURL)
}

type reposList struct {
	// cloneURLToURI records the map from clone URL to repo URI. It is read-only after construction,
	// so does not require synchronization.
	cloneURLToURI map[string]string
}

var _ RepoSource = (*reposList)(nil)

func newReposList(repos []*schema.Repository) *reposList {
	cloneURLToURI := map[string]string{}
	for _, rp := range repos {
		cloneURLToURI[normalizeCloneURL(rp.Url)] = rp.Path
	}
	return &reposList{
		cloneURLToURI: cloneURLToURI,
	}
}

func (c *reposList) CloneURLToRepoURI(cloneURL string) (repoURI api.RepoURI, err error) {
	return api.RepoURI(c.cloneURLToURI[normalizeCloneURL(cloneURL)]), nil
}

// normalizeCloneURL attempts to reduce the cloneURL to a normalized form using some simple
// heuristics. If it finds the heuristics don't apply, it returns the original clone URL.
func normalizeCloneURL(cloneURL string) string {
	var (
		u   *url.URL
		err error
	)
	if strings.HasPrefix(cloneURL, "https://") || strings.HasPrefix(cloneURL, "http://") || strings.HasPrefix(cloneURL, "ssh://") {
		u, err = url.Parse(cloneURL)
	} else { // SCP-like case
		u, err = url.Parse("fake://" + strings.Replace(cloneURL, ":", "/", 1))
	}
	if err != nil {
		return cloneURL
	}
	return fmt.Sprintf("%s/%s", u.Hostname(), strings.TrimPrefix(strings.TrimSuffix(u.Path, ".git"), "/"))
}