package tasks

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/ditashi/jsbeautifier-go/jsbeautifier"
)

// EventListener extracts the functions listening for message events from the DOM
type EventListener struct {
	Event string
}

func (t *EventListener) Priority() uint8 {
	return 1
}

func (t *EventListener) Name() string {
	return fmt.Sprintf("%s Listeners", strings.Title(t.Event))
}

func (t *EventListener) Run(ctx context.Context, url string, absDir string, relDir string) (string, error) {
	f := fmt.Sprintf(`
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
		ret;`, t.Event, t.Event)

	var res map[string]string
	tasks := chromedp.Tasks{
		chromedp.EvaluateAsDevTools(f, &res),
	}
	err := chromedp.Run(ctx, tasks)
	if err != nil {
		return "", err
	}

	// Output
	d := path.Join(absDir, "listeners", t.Event)
	err = os.MkdirAll(d, os.ModePerm)
	if err != nil {
		return "", err
	}

	var html string
	for name, v := range res {
		formatted, err := jsbeautifier.Beautify(&v, jsbeautifier.DefaultOptions())
		if err == nil {
			v = formatted
		}

		// File
		p := path.Join(d, name)
		err = ioutil.WriteFile(p, []byte(v), 0644)
		if err != nil {
			return "", err
		}

		//HTML
		html += fmt.Sprintf("<h5>%s</h5>\n<pre>%s</pre><br><br>", name, v)
	}

	return html, nil
}
