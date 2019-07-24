package tasks

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/chromedp/chromedp"
	"gitlab.com/dcthatch/spydom/config"
)

// The LocalStorage task saves the requested url and final location to files
type LocalStorage struct{}

func (t *LocalStorage) Priority() uint8 {
	return 1
}

func (t *LocalStorage) Slug() string {
	return "localstorage"
}

func (t *LocalStorage) Description() string {
	return "Save the local storage and session storage from the loaded page"
}

func (t *LocalStorage) Init(c *config.Config) error {
	return nil
}

func (t *LocalStorage) Run(ctx context.Context, url string, absDir string, relDir string) error {
	var localStorage string
	var sessionStorage string
	tasks := chromedp.Tasks{
		chromedp.EvaluateAsDevTools("JSON.stringify(localStorage)", &localStorage),
		chromedp.EvaluateAsDevTools("JSON.stringify(sessionStorage)", &sessionStorage),
	}
	if err := chromedp.Run(ctx, tasks); err != nil {
		return fmt.Errorf("failed to retrieve local storage: %v", err)
	}

	lf := path.Join(absDir, "localstorage.txt")
	sf := path.Join(absDir, "sessionstorage.txt")
	localStorage = fmt.Sprintf("%s\n", localStorage)
	sessionStorage = fmt.Sprintf("%s\n", sessionStorage)
	if err := ioutil.WriteFile(lf, []byte(localStorage), 0644); err != nil {
		return fmt.Errorf("failed to write localstorage to file: %v", err)
	}
	if err := ioutil.WriteFile(sf, []byte(sessionStorage), 0644); err != nil {
		return fmt.Errorf("failed to write sessionstorage to file: %v", err)
	}

	return nil
}
