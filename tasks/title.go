package tasks

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/chromedp/chromedp"
	"github.com/danielthatcher/spydom/config"
)

// The Title task saves the requested url and final location to files
type Title struct{}

func (t *Title) Priority() uint8 {
	return 1
}

func (t *Title) Slug() string {
	return "title"
}

func (t *Title) Description() string {
	return "Save the title of the loaded page"
}

func (t *Title) Init(c *config.Config) error {
	return nil
}

func (t *Title) Run(ctx context.Context, url string, absDir string, relDir string) error {
	var title string
	tasks := chromedp.Tasks{chromedp.Title(&title)}
	if err := chromedp.Run(ctx, tasks); err != nil {
		return fmt.Errorf("failed to retrieve final url: %v", err)
	}

	f := path.Join(absDir, "title.txt")
	title = fmt.Sprintf("%s\n", title)
	if err := ioutil.WriteFile(f, []byte(title), 0644); err != nil {
		return fmt.Errorf("failed to save title to file: %v", err)
	}

	return nil
}
