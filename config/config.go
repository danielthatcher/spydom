package config

import "time"

// Config stores the program's config, allowing it to be passed to workers
type Config struct {
	NumThreads int
	Wait       time.Duration
	OutDir     string
	Retries    int
	Verbose    bool
	Timeout    time.Duration
	Enabled    []string
	Disabled   []string
	JS         string
	JSFile     string
	JSPriority uint8
}
