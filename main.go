package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
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
	preserveDir bool
	tasks       []Task
	wait        time.Duration
	wg          *sync.WaitGroup
}

// Load naviagates to the given URL, and waits for the page to load
func (w *Worker) Load(u string) error {
	tasks := chromedp.Tasks{
		chromedp.Navigate(u),
	}
	err := chromedp.Run(*w.ctx, tasks)
	if err == nil {
		time.Sleep(w.wait)
	}
	return err
}

// Work reads URLs from the given channel, loads them, and then performs any
// tasks on the loaded page.
func (w *Worker) Work(urlsChan <-chan string, errorChan chan<- error, reportChan chan<- string) {
	for {
		u, more := <-urlsChan
		if !more {
			w.wg.Done()
			return
		}

		// Output dir
		relDir := strings.Replace(u, "://", "-", 1)
		absDir := path.Join(w.dir, relDir)
		reportFile := path.Join(absDir, "report.html")
		if w.preserveDir {
			// Skip running again if the report file already exists
			b, err := ioutil.ReadFile(reportFile)
			if err == nil {
				reportChan <- string(b)
				continue
			}
		}
		os.MkdirAll(absDir, os.ModePerm)

		err := w.Load(u)
		if err != nil {
			errorChan <- fmt.Errorf("failed to load %v: %v", u, err)
			reportChan <- "Failed to load"
			continue
		}

		// Run all workers on page and output html to reportChan
		fullHTML := fmt.Sprintf("<a href='%s'><h2 align='center'>%s</h2></a><br>", u, u)
		for i := uint8(1); i <= 3; i++ {
			for _, t := range w.tasks {
				if t.Priority() == i {
					html, err := t.Run(*w.ctx, u, absDir, relDir)
					if err != nil {
						errorChan <- fmt.Errorf("failed to run task: %v", err)
					}
					fullHTML += "<h4>" + t.Name() + "</h4><br>" + html + "<br>"
				}
			}
		}
		ioutil.WriteFile(reportFile, []byte(fullHTML), os.ModePerm)
		reportChan <- fullHTML
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
	preserverDir := flag.BoolP("preserve-output", "", false, "If the directory for a host exists, use the existing results instead of overwriting with new output")
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
	reportChan := make(chan string)
	workerWg := &sync.WaitGroup{}
	workerWg.Add(*numThreads)
	workers := make([]*Worker, *numThreads)
	tasks := getTasks()
	for i := range workers {
		ctx, cancel := chromedp.NewContext(context.Background())
		if err := chromedp.Run(ctx); err != nil {
			log.Fatalf("Failed to launch chrome insance: %v\n", err)
		}
		defer cancel()

		w := &Worker{
			ctx:         &ctx,
			dir:         dir,
			preserveDir: *preserverDir,
			wg:          workerWg,
			tasks:       tasks,
			wait:        *wait,
		}
		workers[i] = w
		go w.Work(urlsChan, errorChan, reportChan)
	}

	// Read targets line by line and dispatch to workers
	tfile, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("Failed to open targets file: %v\n", err)
	}
	defer tfile.Close()

	tscanner := bufio.NewScanner(tfile)
	re := regexp.MustCompile("^https?://")
	reportWg := &sync.WaitGroup{}
	go func() {
		defer close(urlsChan)
		for tscanner.Scan() {
			u := tscanner.Text()
			reportWg.Add(1)
			if !re.MatchString(u) {
				u = "https://" + u
			}

			log.Println(u)
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

	// Generate the report
	var reportHTML string
	go func() {
		for {
			row := <-reportChan
			reportHTML += row + "<br><br>"
			reportWg.Done()
		}
	}()

	workerWg.Wait()
	reportWg.Wait()

	start := `<html><head><title>SpyDOM Report</title></head><body>`
	end := `</body></html>`
	reportHTML = start + reportHTML + end
	ioutil.WriteFile(path.Join(dir, "report.html"), []byte(reportHTML), 0644)
}
