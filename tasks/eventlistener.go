package tasks

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/chromedp/chromedp"
	"github.com/danielthatcher/spydom/config"
	"github.com/ditashi/jsbeautifier-go/jsbeautifier"
)

// EventListener extracts the functions listening for message events from the DOM
type EventListener struct {
	Event string
}

func (t *EventListener) Priority() uint8 {
	return 1
}

func (t *EventListener) Slug() string {
	return t.Event
}

func (t *EventListener) Description() string {
	return fmt.Sprintf("Extract all %s event listeners from the page", t.Event)
}

func (t *EventListener) Init(c *config.Config) error {
	return nil
}

func (t *EventListener) Run(ctx context.Context, url string, absDir string, relDir string) error {
	f := fmt.Sprintf(`
		(function(){
			let nodes = [window, document];
			let elements = document.querySelectorAll("*");
			nodes.push.apply(nodes, elements);
			let listeners = {};
			nodes.forEach(function(e) { listeners = Object.assign({}, listeners, getEventListeners(e)); } );
			let ret = {};
			if ("%s" in listeners) {
				listeners = listeners["%s"];
				listeners.forEach(function(e) {
					let name = e.listener.name;
					let x = 1;
					while (name in listeners) {
						name = e.listener.name + x.toString();
						x += 1;
					}
					if (name == "") {
						name = "unnamed";
					}
					ret[name] = e.listener.toString();
				});
			}
			return ret;
		})()`, t.Event, t.Event)

	var res map[string]string
	tasks := chromedp.Tasks{
		chromedp.EvaluateAsDevTools(f, &res),
	}
	err := chromedp.Run(ctx, tasks)
	if err != nil {
		return err
	}

	// Output
	d := path.Join(absDir, "listeners", t.Event)
	err = os.MkdirAll(d, os.ModePerm)
	if err != nil {
		return err
	}

	for name, v := range res {
		formatted, _ := jsbeautifier.Beautify(&v, jsbeautifier.DefaultOptions())

		// Write original to file
		p := path.Join(d, name)
		if err := ioutil.WriteFile(p, []byte(v), 0644); err != nil {
			return err
		}

		// Write beautified version to file
		p = path.Join(d, fmt.Sprintf("%s.beautified", name))
		if err := ioutil.WriteFile(p, []byte(formatted), 0644); err != nil {
			return err
		}
	}

	return nil
}
