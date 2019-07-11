package tasks

import (
	"context"
	"io/ioutil"
	"math"
	"path"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type Screenshot struct{}

func (t *Screenshot) Priority() uint8 {
	return 1
}

func (t *Screenshot) Name() string {
	return "Screenshot"
}

func (t *Screenshot) Run(ctx context.Context, url string, absDir string, relDir string) error {
	var buf []byte
	tasks := chromedp.Tasks{chromedp.ActionFunc(func(ctx context.Context) error {
		_, _, contentSize, err := page.GetLayoutMetrics().Do(ctx)
		if err != nil {
			return err
		}

		w := int64(math.Ceil(contentSize.Width))
		h := int64(math.Ceil(contentSize.Height))
		if w <= 0 || h <= 0 {
			return nil
		}
		err = emulation.SetDeviceMetricsOverride(w, h, 1, false).
			WithScreenOrientation(&emulation.ScreenOrientation{
				Type:  emulation.OrientationTypePortraitPrimary,
				Angle: 0,
			}).Do(ctx)

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
