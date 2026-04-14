package github

import (
	"github.com/c0de-ch/lab-githubapp/internal/store"
)

type Client struct {
	auth  *AuthProvider
	store store.Store
}

func NewClient(auth *AuthProvider, s store.Store) *Client {
	return &Client{
		auth:  auth,
		store: s,
	}
}

func (c *Client) clientForRepo(owner, repo string) (*AuthProvider, int64, error) {
	inst, err := c.store.GetInstallationForRepo(owner, repo)
	if err != nil {
		return nil, 0, err
	}
	return c.auth, inst.ID, nil
}
