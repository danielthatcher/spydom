package main

import (
	"context"
	"io/ioutil"
	"path"

	"github.com/chromedp/chromedp"
)

// Task represents a task that should be performed on all pages
type Task interface {
	// The Priority of a task determines when it will be run.
	// Tasks with priority 1 do passive checks that don't modify the DOM, and are run first.
	// Tasks with priority 2 do light active checks that make largely inconsequential modifications to the DOM.
	// Tasks with priority 3 may make significant changes to the DOM, that might interfere with other tasks.
	Priority() uint8

	// Run runs the task, saving the results in the given directory
	Run(ctx context.Context, url string, dir string, c *chromedp.Res) error
}

func getTasks() []Task {
	return []Task{
		&taskScreenshot{},
	}
}

type taskScreenshot struct{}

func (t *taskScreenshot) Priority() uint8 {
	return 1
}

func (t *taskScreenshot) Run(ctx context.Context, url string, dir string, c *chromedp.Res) error {
	var buf []byte
	tasks := chromedp.Tasks{
		chromedp.CaptureScreenshot(&buf),
	}
	err := c.Run(ctx, tasks)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(dir, "screenshot.png"), buf, 0644)
	return err
}
