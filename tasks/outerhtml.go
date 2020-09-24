package tasks

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/danielthatcher/spydom/config"
)

// The OuterHTML task saves the outer HTML of the rendered page
type OuterHTML struct{}

func (t *OuterHTML) Priority() uint8 {
	return 1
}

func (t *OuterHTML) Slug() string {
	return "outerhtml"
}

func (t *OuterHTML) Description() string {
	return "Save the outer HTML of the rendered page."
}

func (t *OuterHTML) Init(c *config.Config) error {
	return nil
}

func (t *OuterHTML) Run(ctx context.Context, url string, absDir string, relDir string) error {
	var html string
	tasks := chromedp.Tasks{chromedp.ActionFunc(func(c context.Context) error {
		node, err := dom.GetDocument().Do(c)
		if err != nil {
			return err
		}
		html, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(c)
		return err
	})}

	if err := chromedp.Run(ctx, tasks); err != nil {
		return fmt.Errorf("failed to retrieve outer HTML: %v", err)
	}

	f := path.Join(absDir, "outerhtml.txt")
	html = fmt.Sprintf("%s\n", html)
	if err := ioutil.WriteFile(f, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write outer HTML to file: %v", err)
	}

	return nil
}
