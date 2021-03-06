package tasks

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/chromedp/chromedp"
	"github.com/danielthatcher/spydom/config"
)

type JSRunner struct {
	script   string
	priority uint8
}

func (t *JSRunner) Priority() uint8 {
	return t.priority
}

func (t *JSRunner) Slug() string {
	return "jsrunner"
}

func (t *JSRunner) Description() string {
	return "Run custom JavaScript on the page. JavaScript can be supplied directly with the --js flag, or from the file given by the --js-file flag. This module is only enabled if the --js or --js-file flags are specified. Values are returned from variables in string format by putting just the variable as a statement, e.g. 'x=document.domain; x'."
}

func (t *JSRunner) Init(c *config.Config) error {
	if c.JSPriority < 0 || c.JSPriority > 4 {
		return fmt.Errorf("priority must be between 0 and 4")
	}
	t.priority = c.JSPriority

	if c.JS != "" {
		t.script = c.JS
		return nil
	}

	if c.JSFile != "" {
		buf, err := ioutil.ReadFile(c.JSFile)
		if err != nil {
			return fmt.Errorf("failed to read JS file: %v", err)
		}
		t.script = string(buf)
		return nil
	}

	return fmt.Errorf("no JavaScript specified")
}

func (t *JSRunner) Run(ctx context.Context, url string, absDir string, relDir string) error {
	var res string
	tasks := chromedp.Tasks{
		chromedp.EvaluateAsDevTools(t.script, &res),
	}
	err := chromedp.Run(ctx, tasks)
	if err != nil {
		return fmt.Errorf("failed to run custom JavaScript: %v", err)
	}

	res = fmt.Sprintf("%s\n", res)
	f := path.Join(absDir, "jsrunner.txt")
	if err = ioutil.WriteFile(f, []byte(res), 0644); err != nil {
		return err
	}

	return nil
}
