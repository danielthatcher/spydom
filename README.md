# Spydom
Spydom is a scanner that automates Chrome headless to load a list of web pages. Information such as `postmessage` and `hashchange` listeners and generated HTML, is then extracted from these pages, and tasks such as taking screenshots are performed against each page.

## Usage
For its most basic usage, spydom can be be run using
```bash
spydom targets.txt
```
where `targets.txt` is a file containing a list of URLs, one per line. This will run all of spydom's default [modules](#modules) against each page, and generate an HTML report displaying the results. Each module also saves its output to the filesystem to make processing by other tools easy.

You can view the help text with `spydom -h`.

## Installation
As spydom relies on browser automation through the [chromedp](https://github.com/chromedp/chromedp) library you will need to install a browser that is compatible with the Chrome DevTools protocol, such as Chrome or Chromium.

Then, with a properly configured `$GOPATH`, you can run
```bash
go get -v github.com/danielthatcher/spydom
```
to install spydom.

## Modules
The following modules are enabled by default:

Module | Description
-|-
screenshot|Take a screenshot of the page
message	|Extract all message event listeners from the page
hashchange|Extract all hashchange event listeners from the page
location|Save the requested URL and the final URL of the loaded page
localstorage|Save the local storage and session storage from the loaded page
outerhtml|Save the outer HTML of the rendered page.
title	|Save the title of the loaded page
heapsnapshot|Save a snapshot of the JavaScript heap

### The jsrunner module
The `jsrunnner` module allows you to run a custom snippet of JavaScript on each page and retrieve the result. JavaScript can be specified directly on the command line with the `--js` flag, or from a file with the `--js-file` flag. This module is disabled unless one of these flags is specified.

To receive output from the JavaScript snippet, include a variable containing the output as the final statement of the snippet. For example, to retrieve `document.domain` from every page, you would run
```bash
spydom --js='x=document.domain; x' targets.txt
```

### Enabling and disabling modules
Modules can be enabled and disabled with the `-e` and `-d` flags respectively. These flags can be specified multiple times to enable or disable multiple modules.

To enable just the `title` and `location` modules, you would run
```bash
spydom -e title -e location targets.txt
```

To run all the default modules apart from the `heapsnapshot` module you would run
```bash
spydom -d heapsnapshot targets.txt
```

## Reporting
By default, spydom will store all its output in a directory named `spydom_output`. This includes a directory for each URL loaded in which the plain text output from each module will be stored, as well as a `report.html` file which is a standalone file detailing the results of the scan. The report file groups pages by their final URL after all redirections, so pages that redirect to the same location will be grouped.

## Future work
### New modules
This tool can always benefit from more modules. Below is a list of modules I believe will benefit the tool and intend to add at some point, though if you have any other modules you would like to see then please feel free to open a pull request or submit an issue.

- DOM event logger
- websockets message logger
- Response cookies
- Loaded JavaScript files
- Response headers

### Passively recording data from an existing Chrome session
spydom currently only acts as a scanner, automating a browser to load pages and then running modules against those pages. It would also be possible to have spydom attach to the remote debugging port of an existing Chrome session in order to run modules against each page a user loads.