package main

import (
	"context"

	"gitlab.com/dcthatch/spydom/tasks"
)

// Task represents a task that should be performed on all pages
type Task interface {
	// The Priority of a task determines when it will be run.
	// Tasks with priority 1 do passive checks that don't modify the DOM, and are run first.
	// Tasks with priority 2 do light active checks that make largely inconsequential modifications to the DOM.
	// Tasks with priority 3 may make significant changes to the DOM, that might interfere with other tasks.
	Priority() uint8

	// Run runs the task, saving the results in the given directory and returning the HTML to display those results.
	Run(ctx context.Context, url string, absDir string, relDir string) error

	// Name returns the name of the plugin that should be used for reporting
	Name() string
}

func getTasks() []Task {
	return []Task{
		&tasks.Screenshot{},
		&tasks.EventListener{Event: "message"},
	}
}
