package github

import (
	"net/http"
	"sync"

	"github.com/bradleyfalzon/ghinstallation/v2"
	ghlib "github.com/google/go-github/v68/github"
)

type AuthProvider struct {
	appsTransport *ghinstallation.AppsTransport
	mu            sync.RWMutex
	transports    map[int64]*ghinstallation.Transport
}

func NewAuthProvider(appID int64, privateKey []byte) (*AuthProvider, error) {
	appsTransport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, appID, privateKey)
	if err != nil {
		return nil, err
	}
	return &AuthProvider{
		appsTransport: appsTransport,
		transports:    make(map[int64]*ghinstallation.Transport),
	}, nil
}

func (a *AuthProvider) AppClient() *ghlib.Client {
	return ghlib.NewClient(&http.Client{Transport: a.appsTransport})
}

func (a *AuthProvider) ClientForInstallation(installationID int64) *ghlib.Client {
	a.mu.RLock()
	tr, ok := a.transports[installationID]
	a.mu.RUnlock()

	if ok {
		return ghlib.NewClient(&http.Client{Transport: tr})
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after acquiring write lock
	if tr, ok = a.transports[installationID]; ok {
		return ghlib.NewClient(&http.Client{Transport: tr})
	}

	tr = ghinstallation.NewFromAppsTransport(a.appsTransport, installationID)
	a.transports[installationID] = tr
	return ghlib.NewClient(&http.Client{Transport: tr})
}
