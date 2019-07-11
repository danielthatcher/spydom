package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	flag "github.com/spf13/pflag"
)

// Worker represents the tasks for a thread
type Worker struct {
	ctx         *context.Context
	dir         string
	id          int
	preserveDir bool
	retries     int
	tasks       []Task
	timeout     time.Duration
	verbose     bool
	wait        time.Duration
	wg          *sync.WaitGroup
}

// Load naviagates to the given URL, and waits for the page to load
func (w *Worker) Load(u string) error {
	if w.verbose {
		log.Printf("Worker %d: loading %s\n", w.id, u)
	}
	tasks := chromedp.Tasks{
		chromedp.Navigate(u),
	}
	ctx, cancel := context.WithTimeout(*w.ctx, w.timeout)
	defer cancel()
	err := chromedp.Run(ctx, tasks)
	if err == nil {
		time.Sleep(w.wait)
		if w.verbose {
			log.Printf("Worker %d: loaded %s\n", w.id, u)
		}
	}
	return err
}

// Work reads URLs from the given channel, loads them, and then performs any
// tasks on the loaded page.
func (w *Worker) Work(urlsChan <-chan string, errorChan chan<- error) {
	for {
		u, more := <-urlsChan
		if !more {
			w.wg.Done()
			return
		}

		// Output dir
		relDir := strings.Replace(u, "://", "-", 1)
		absDir := path.Join(w.dir, relDir)
		os.MkdirAll(absDir, os.ModePerm)

		attempt := 1
		err := w.Load(u)
		success := true
		for err != nil {
			if attempt >= w.retries {
				errorChan <- fmt.Errorf("failed to load %v: %v; giving up after %d attempts", u, err, attempt)
				success = false
				break
			}

			attempt++
			errorChan <- fmt.Errorf("failed to load %v: %v; retrying (%d/%d)", u, err, attempt, w.retries)
			err = w.Load(u)
		}

		if !success {
			continue
		}

		// Run all workers on page
		for i := uint8(1); i <= 3; i++ {
			for _, t := range w.tasks {
				if t.Priority() == i {
					err := t.Run(*w.ctx, u, absDir, relDir)
					if err != nil {
						errorChan <- fmt.Errorf("failed to run task: %v", err)
					}
				}
			}
		}
	}
}

func main() {
	// Argument parsing
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s [OPTIONS]... [TARGETS FILE]\n", os.Args[0])
		flag.PrintDefaults()
	}

	numThreads := flag.IntP("threads", "t", 10, "Number of threads to run")
	wait := flag.DurationP("wait", "w", 2*time.Second, "Number of milliseconds to wait for page to load before running tasks")
	relDir := flag.StringP("output", "o", "spydom_output", "The directory to store output in")
	retries := flag.IntP("retries", "r", 3, "Maximum number of times to load earch URL when encountering errors")
	verbose := flag.BoolP("verbose", "v", false, "Use verbose output")
	timeout := flag.DurationP("timeout", "", 10*time.Second, "The time to allow for all tasks to be run on a page before giving up")
	flag.Parse()

	if flag.NArg() != 1 || flag.Arg(0) == "" {
		fmt.Println("Please supply a targets file")
		flag.Usage()
		os.Exit(1)
	}
	dir, err := filepath.Abs(*relDir)
	if err != nil {
		log.Fatalf("Failed to open output directory: %v\n", err)
	}

	urlsChan := make(chan string)
	errorChan := make(chan error)
	workerWg := &sync.WaitGroup{}
	workerWg.Add(*numThreads)
	workers := make([]*Worker, *numThreads)
	tasks := getTasks()
	var ctx context.Context
	for i := range workers {
		var cancel context.CancelFunc
		if ctx == nil {
			ctx, cancel = chromedp.NewContext(context.Background())
		} else {
			ctx, cancel = chromedp.NewContext(ctx)
		}
		if err := chromedp.Run(ctx); err != nil {
			log.Fatalf("Failed to launch chrome insance: %v\n", err)
		}
		defer cancel()

		w := &Worker{
			ctx:     &ctx,
			dir:     dir,
			id:      i,
			retries: *retries,
			tasks:   tasks,
			timeout: *timeout,
			verbose: *verbose,
			wait:    *wait,
			wg:      workerWg,
		}
		workers[i] = w
		go w.Work(urlsChan, errorChan)
	}

	// Read targets line by line and dispatch to workers
	tfile, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("Failed to open targets file: %v\n", err)
	}
	defer tfile.Close()

	tscanner := bufio.NewScanner(tfile)
	re := regexp.MustCompile("^https?://")
	go func() {
		defer close(urlsChan)
		for tscanner.Scan() {
			u := tscanner.Text()
			if !re.MatchString(u) {
				u = "https://" + u
			}
			urlsChan <- u
		}

		if err = tscanner.Err(); err != nil {
			log.Fatalf("Error while reading targets file: %v\n", err)
		}
	}()

	// Report errors to stderr
	go func() {
		l := log.New(os.Stderr, "ERROR: ", 0)
		for {
			err := <-errorChan
			l.Println(err)
		}
	}()

	workerWg.Wait()
}
