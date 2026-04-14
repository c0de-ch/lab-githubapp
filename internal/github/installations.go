package github

import (
	"context"

	"github.com/c0de-ch/lab-githubapp/internal/store"
	ghlib "github.com/google/go-github/v68/github"
)

func (c *Client) SyncInstallations(ctx context.Context) error {
	client := c.auth.AppClient()

	opts := &ghlib.ListOptions{PerPage: 100}
	for {
		installations, resp, err := client.Apps.ListInstallations(ctx, opts)
		if err != nil {
			return err
		}

		for _, inst := range installations {
			installation := &store.Installation{
				ID:           inst.GetID(),
				AppID:        inst.GetAppID(),
				AccountLogin: inst.GetAccount().GetLogin(),
				AccountType:  inst.GetAccount().GetType(),
				TargetType:   inst.GetTargetType(),
			}
			if err := c.store.UpsertInstallation(installation); err != nil {
				return err
			}

			if err := c.syncInstallationRepos(ctx, inst.GetID()); err != nil {
				return err
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil
}

func (c *Client) syncInstallationRepos(ctx context.Context, installationID int64) error {
	client := c.auth.ClientForInstallation(installationID)

	opts := &ghlib.ListOptions{PerPage: 100}
	for {
		result, resp, err := client.Apps.ListRepos(ctx, opts)
		if err != nil {
			return err
		}

		for _, r := range result.Repositories {
			repo := &store.Repository{
				ID:             r.GetID(),
				InstallationID: installationID,
				Owner:          r.GetOwner().GetLogin(),
				Name:           r.GetName(),
				FullName:       r.GetFullName(),
				Private:        r.GetPrivate(),
				HTMLURL:        r.GetHTMLURL(),
			}
			if err := c.store.UpsertRepository(repo); err != nil {
				return err
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil
}
