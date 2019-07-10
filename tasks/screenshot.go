package tasks

import (
	"context"
	"io/ioutil"
	"path"

	"github.com/chromedp/chromedp"
)

type Screenshot struct{}

func (t *Screenshot) Priority() uint8 {
	return 1
}

func (t *Screenshot) Name() string {
	return "Screenshot"
}

func (t *Screenshot) Run(ctx context.Context, url string, absDir string, relDir string) (string, error) {
	var buf []byte
	tasks := chromedp.Tasks{
		chromedp.CaptureScreenshot(&buf),
	}
	err := chromedp.Run(ctx, tasks)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(path.Join(absDir, "screenshot.png"), buf, 0644)
	if err != nil {
		return "", err
	}

	html := "<img src='" + path.Join(relDir, "screenshot.png") + "' />"
	return html, nil
}
