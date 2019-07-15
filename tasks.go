package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"gitlab.com/dcthatch/spydom/config"
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

	// Slug returns the command-line friendly name that is used to enable or disable the module
	Slug() string

	// Description returns a description of the task
	Description() string

	// Init takes a Config object and initialises the task
	Init(c *config.Config) error
}

func allTasks() []Task {
	return []Task{
		&tasks.Screenshot{},
		&tasks.EventListener{Event: "message"},
		&tasks.EventListener{Event: "hashchange"},
		&tasks.Location{},
		&tasks.JSRunner{},
	}

}

func listTasks() {
	tasks := allTasks()

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintln(w, "Name\t\tDescription")
	for _, t := range tasks {
		fmt.Fprintf(w, "%v\t%s\n", t.Slug(), t.Description())
	}
	w.Flush()
}

func getTasks(c *config.Config) ([]Task, error) {
	tasks := allTasks()
	for i := range tasks {
		tasks[i].Init(c)
	}

	if c.Enabled != nil {
		newTasks := []Task{}
		for _, slug := range c.Enabled {
			for _, t := range tasks {
				if t.Slug() == slug {
					newTasks = append(newTasks, t)
					break
				}
			}
		}
		tasks = newTasks
	}

	// Disable the jsrunner module if --js or --js-file are not specified
	if c.JS == "" && c.JSFile == "" {
		if c.Disabled == nil {
			c.Disabled = []string{"jsrunner"}
		} else {
			c.Disabled = append(c.Disabled, "jsrunner")
		}
	}

	if c.Disabled != nil {
		newTasks := []Task{}
		for _, t := range tasks {
			enabled := true
			for _, slug := range c.Disabled {
				if t.Slug() == slug {
					enabled = false
					break
				}
			}

			if enabled {
				newTasks = append(newTasks, t)
			}
		}
		tasks = newTasks
	}

	return tasks, nil
}
