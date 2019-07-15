package tasks

import (
	"context"
	"io/ioutil"
	"path"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"gitlab.com/dcthatch/spydom/config"
)

type Screenshot struct {
	width  int
	height int
}

func (t *Screenshot) Priority() uint8 {
	return 1
}

func (t *Screenshot) Name() string {
	return "Screenshot"
}

func (t *Screenshot) Slug() string {
	return "screenshot"
}

func (t *Screenshot) Description() string {
	return "Take a screenshot of the page"
}

func (t *Screenshot) Init(c *config.Config) {
}

func (t *Screenshot) Run(ctx context.Context, url string, absDir string, relDir string) error {
	var buf []byte
	tasks := chromedp.Tasks{chromedp.ActionFunc(func(ctx context.Context) error {
		_, _, contentSize, err := page.GetLayoutMetrics().Do(ctx)
		if err != nil {
			return err
		}

		buf, err = page.CaptureScreenshot().
			WithClip(&page.Viewport{
				X:      contentSize.X,
				Y:      contentSize.Y,
				Width:  contentSize.Width,
				Height: contentSize.Height,
				Scale:  1,
			}).Do(ctx)

		if err != nil {
			return err
		}

		return nil
	})}

	err := chromedp.Run(ctx, tasks)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(absDir, "screenshot.png"), buf, 0644)
	if err != nil {
		return err
	}

	return nil
}
