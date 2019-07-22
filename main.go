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

	"github.com/chromedp/cdproto/security"
	"github.com/chromedp/chromedp"
	flag "github.com/spf13/pflag"
	"gitlab.com/dcthatch/spydom/config"
)

// Worker represents the tasks for a thread
type Worker struct {
	ctx    *context.Context
	id     int
	tasks  []Task
	wg     *sync.WaitGroup
	config *config.Config
}

// Load naviagates to the given URL, and waits for the page to load
func (w *Worker) Load(u string) error {
	if w.config.Verbose {
		log.Printf("Worker %d: loading %s\n", w.id, u)
	}
	tasks := chromedp.Tasks{
		chromedp.Navigate(u),
	}
	ctx, cancel := context.WithTimeout(*w.ctx, w.config.Timeout)
	defer cancel()
	err := chromedp.Run(ctx, tasks)
	if err == nil {
		time.Sleep(w.config.Wait)
		if w.config.Verbose {
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
		absDir := path.Join(w.config.OutDir, relDir)
		os.MkdirAll(absDir, os.ModePerm)

		attempt := 1
		err := w.Load(u)
		success := true
		for err != nil {
			if attempt >= w.config.Retries {
				errorChan <- fmt.Errorf("worker %d: failed to load %v: %v; giving up after %d attempts", w.id, u, err, attempt)
				success = false
				break
			}

			attempt++
			errorChan <- fmt.Errorf("worker %d: failed to load %v: %v; retrying (%d/%d)", w.id, u, err, attempt, w.config.Retries)
			err = w.Load(u)
		}

		if !success {
			continue
		}

		// Run all workers on page. Start at 0 and go to 4 in as these are valid
		// priorities for the jsrunner module
		for i := uint8(0); i <= 4; i++ {
			for _, t := range w.tasks {
				if t.Priority() == i {
					err := t.Run(*w.ctx, u, absDir, relDir)
					if err != nil {
						errorChan <- fmt.Errorf("failed to run task %v: %v", t.Slug(), err)
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

	conf := config.Config{}
	var relDir string
	flag.IntVarP(&conf.NumThreads, "threads", "t", 10, "Number of threads to run")
	flag.DurationVarP(&conf.Wait, "wait", "w", 2*time.Second, "Number of milliseconds to wait for page to load before running tasks")
	flag.StringVarP(&relDir, "output", "o", "spydom_output", "The directory to store output in")
	flag.IntVarP(&conf.Retries, "retries", "r", 3, "Maximum number of times to load earch URL when encountering errors")
	flag.BoolVarP(&conf.Verbose, "verbose", "v", false, "Use verbose output")
	flag.DurationVarP(&conf.Timeout, "timeout", "", 10*time.Second, "The time to allow for all tasks to be run on a page before giving up")
	flag.StringSliceVarP(&conf.Enabled, "enable", "e", nil, "Enable only the specified modules")
	flag.StringSliceVarP(&conf.Disabled, "disable", "d", nil, "Disable these modules")

	flag.StringVarP(&conf.JS, "js", "", "", "JavaScript to run with the jsrunner module")
	flag.StringVarP(&conf.JSFile, "js-file", "", "", "A file containing JavaScript to run with the jsrunner module")
	flag.Uint8VarP(&conf.JSPriority, "js-priority", "", 4, "The run priority for the jsrunner module, between 0 and 4. Modules with lower priorities get run sooner.")

	ls := flag.BoolP("list-tasks", "l", false, "List tasks and exit")
	insecure := flag.BoolP("insecure", "k", false, "Ignore certificate errors")
	visible := flag.BoolP("visible", "", false, "Show the Chrome window rather than running in headless mode")
	flag.Parse()

	if *ls {
		listTasks()
		os.Exit(0)
	}

	if flag.NArg() != 1 || flag.Arg(0) == "" {
		fmt.Println("Please supply a targets file")
		flag.Usage()
		os.Exit(1)
	}

	dir, err := filepath.Abs(relDir)
	if err != nil {
		log.Fatalf("Failed to open output directory: %v\n", err)
	}
	conf.OutDir = dir

	// User options controlling chrome
	certParams := security.SetIgnoreCertificateErrors(*insecure)
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", !*visible))

	tasks, err := getTasks(&conf)
	if err != nil {
		log.Fatal(err)
	}
	urlsChan := make(chan string)
	errorChan := make(chan error)
	workerWg := &sync.WaitGroup{}
	workerWg.Add(conf.NumThreads)
	workers := make([]*Worker, conf.NumThreads)
	var ctx *context.Context
	for i := range workers {
		var cancel context.CancelFunc
		var newCtx context.Context
		if ctx == nil {
			newCtx, cancel = chromedp.NewExecAllocator(context.Background(), opts...)
			newCtx, cancel = chromedp.NewContext(newCtx)
		} else {
			newCtx, cancel = chromedp.NewContext(*ctx)
		}
		if err := chromedp.Run(newCtx, certParams); err != nil {
			log.Fatalf("Failed to launch chrome insance: %v\n", err)
		}
		defer cancel()

		ctx = &newCtx
		w := &Worker{
			ctx:    &newCtx,
			id:     i,
			tasks:  tasks,
			wg:     workerWg,
			config: &conf,
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
