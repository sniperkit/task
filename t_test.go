package task

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// Worker is a simple worker that GETs web pages sequenceally.
type Worker struct {
	Stop     chan *T           // Channel to stop the event loop
	Gets     chan *GetTask     // Single GET requests
	BulkGets chan *BulkGetTask // Multiple GET Requests
}

// Single GET request task
type GetTask struct {
	T
	URL      string // URL of the web page to GET
	Response string // Response from the worker
}

func NewGetTask(url string) *GetTask {
	return &GetTask{*New(), url, ""}
}

// Multiple GET requests in a single task
type BulkGetTask struct {
	T
	URLs      []string      // The list of URLs to GET
	Responses []string      // Responses from the worker for each page
	Interval  time.Duration // Not to issue main requests in a sthort time
}

func NewBulkGetTask(urls []string, interval time.Duration) *BulkGetTask {
	return &BulkGetTask{*New(), urls, make([]string, len(urls)), interval}
}

func (w *Worker) Start() {
	for {
		select {
		case t := <-w.Stop:
			if t.Start() { // Check if the request has been canceled

				/*
				 There could be some time consuming work to release resources
				*/

				t.Done() // Notify that it's actually stopped
				break    // Break the event loop
			}

			// If the requester has changed its mined, just do nothing

		case t := <-w.Gets: // Single GET request
			if !t.Start() {
				continue // Ignore canceled tasks
			}

			t.Response, t.Error = httpGet(t.URL) // t.Error is always available
			t.Done()

		case t := <-w.BulkGets: // Multiple GET requests
			if !t.Start() {
				continue
			}

			timer := time.NewTimer(t.Interval)

			for i, url := range t.URLs {
				select {
				case <-t.AbortingC(): // You can abort immediately once get a notification
					t.Error = errors.New("Task aborted")
					t.Done()
					continue

				case <-timer.C: // Wait for some time not to overload remote servers
					resp, err := httpGet(url)
					if err != nil {
						t.Error = err
						t.Done()
						continue
					}

					t.Responses[i] = resp
					timer.Reset(t.Interval)
				}
			}

			t.Done()
		}
	}
}

func httpGet(url string) (body string, err error) {
	time.Sleep(time.Second) // It takes some time...
	return fmt.Sprintf("CONTENTS OF %s", url), nil
}

func TestT(test *testing.T) {
	worker := &Worker{
		Stop:     make(chan *T, 1),
		Gets:     make(chan *GetTask, 10),
		BulkGets: make(chan *BulkGetTask, 10),
	}

	go worker.Start()

	// Simply request a page
	req0 := NewGetTask("http://github.com/")
	worker.Gets <- req0
	req0.WaitDone()
	fmt.Println(req0.Response)
	// CONTENTS OF http://github.com/

	// Send a bulk request
	req1 := NewBulkGetTask([]string{"http://github.com/", "http://github.com/yudai"}, time.Second)
	worker.BulkGets <- req1
	req1.WaitDone()
	fmt.Println(req1.Responses[0])
	fmt.Println(req1.Responses[1])
	// CONTENTS OF http://github.com/
	// CONTENTS OF http://github.com/yudai

	// Request but abort
	req2 := NewGetTask("http://github.com/")
	req3 := NewGetTask("http://github.com/yudai")
	worker.Gets <- req2
	worker.Gets <- req3
	ok := req3.Cancel()
	if ok {
		// Succeeded to cancel
	}

	// Abort a task being processed
	req4 := NewBulkGetTask(
		[]string{
			"http://github.com/",
			"http://github.com/yudai",
			"http://github.com/yudai/task",
		}, time.Second)
	worker.BulkGets <- req4
	time.Sleep(time.Second * 3)
	req4.Abort()
	req4.WaitDone()
	fmt.Println(req4.Responses)

	// Stop the worker
	stop := New()
	worker.Stop <- stop
	stop.WaitDone()
}
