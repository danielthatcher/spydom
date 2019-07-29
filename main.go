package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
	urlsWg *sync.WaitGroup
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
// tasks on the loaded page. URLs which failed to load are sent down failureChan
func (w *Worker) Work(urlsChan <-chan string, errorChan chan<- error, failureChan chan<- string) {
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

		err := w.Load(u)
		if err != nil {
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
		w.urlsWg.Done()
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

	// Channels to communicate with workers
	// urlsChan is used to send URLs to workers to load and scan
	// errorsChan is used to send URLs from workers
	// failureChan is used to send URLs which failed to load from workers
	urlsChan := make(chan string)
	errorChan := make(chan error)
	failureChan := make(chan string)

	// urlsWg tracks the URLs which have been loaded
	urlsWg := &sync.WaitGroup{}

	// workerWg tracks which workers are finished
	workerWg := &sync.WaitGroup{}
	workerWg.Add(conf.NumThreads)

	// Create the workers
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
			urlsWg: urlsWg,
			config: &conf,
		}
		workers[i] = w
		go w.Work(urlsChan, errorChan, failureChan)
	}

	// Try to open the targets file
	tfile, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("Failed to open targets file: %v\n", err)
	}
	defer tfile.Close()

	// Add to the urlsWg for each line in the file
	tscanner := bufio.NewScanner(tfile)
	countScanner := bufio.NewScanner(tfile)
	for countScanner.Scan() {
		urlsWg.Add(1)
	}

	// Read targets line by line and dispatch to workers
	tfile.Seek(0, io.SeekStart)
	re := regexp.MustCompile("^https?://")
	go func() {
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

	// Retry failure URLs
	retries := make(map[string]int)
	go func() {
		for {
			u := <-failureChan
			retries[u]++
			if retries[u] > conf.Retries {
				log.Printf("Failed to load %s. Giving up after %d tries.\n", u, conf.Retries)
				delete(retries, u)
				continue
			}
			log.Printf("Failed to load %s. Will retry (%d/%d).\n", u, retries[u], conf.Retries)
			urlsChan <- u
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

	urlsWg.Wait()
	close(urlsChan)
	workerWg.Wait()
}
