package tasks

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/chromedp/chromedp"
	"github.com/danielthatcher/spydom/config"
)

// The Location task saves the requested url and final location to files
type Location struct{}

func (t *Location) Priority() uint8 {
	return 1
}

func (t *Location) Slug() string {
	return "location"
}

func (t *Location) Description() string {
	return "Save the requested URL and the final URL of the loaded page"
}

func (t *Location) Init(c *config.Config) error {
	return nil
}

func (t *Location) Run(ctx context.Context, url string, absDir string, relDir string) error {
	var newurl string
	tasks := chromedp.Tasks{chromedp.Location(&newurl)}
	if err := chromedp.Run(ctx, tasks); err != nil {
		return fmt.Errorf("failed to retrieve final url: %v", err)
	}

	of := path.Join(absDir, "requested-url.txt")
	nf := path.Join(absDir, "final-url.txt")
	ourl := fmt.Sprintf("%s\n", url)
	nurl := fmt.Sprintf("%s\n", newurl)
	if err := ioutil.WriteFile(of, []byte(ourl), 0644); err != nil {
		return fmt.Errorf("failed to write original url to file: %v", err)
	}
	if err := ioutil.WriteFile(nf, []byte(nurl), 0644); err != nil {
		return fmt.Errorf("failed to write final url to file: %v", err)
	}

	return nil
}
