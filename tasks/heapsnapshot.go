package tasks

import (
	"context"
	"os"
	"path"

	"github.com/chromedp/cdproto/heapprofiler"
	"github.com/chromedp/chromedp"
	"gitlab.com/dcthatch/spydom/config"
)

// The HeapSnapshot task task saves a snapshot of the heap
type HeapSnapshot struct{}

func (t *HeapSnapshot) Priority() uint8 {
	// While this task doesn't make modifications to the page, it would be good to run after
	// other tasks have run to try and populate the heap a little more
	return 2
}

func (t *HeapSnapshot) Slug() string {
	return "heapsnapshot"
}

func (t *HeapSnapshot) Description() string {
	return "Save a snapshot of the heap"
}

func (t *HeapSnapshot) Init(c *config.Config) error {
	return nil
}

func (t *HeapSnapshot) Run(ctx context.Context, url string, absDir string, relDir string) error {
	outfile := path.Join(absDir, "heapsnapshot")
	// The heap snapshot is returned through events, so wait for those events
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*heapprofiler.EventAddHeapSnapshotChunk); ok {
			go func() {
				c := ev.Chunk
				_ = c
				f, err := os.OpenFile(outfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					// TODO: It would be nice to have a way to report these errors
					return
				}
				f.Write([]byte(c))
			}()
		}
	})
	tasks := chromedp.Tasks{
		heapprofiler.TakeHeapSnapshot(),
	}
	if err := chromedp.Run(ctx, tasks); err != nil {
		return err
	}
	return nil
}
