/*
Package task provides a simple struct T that helps making intaraction with channels and goroutines simpler.
*/
package task

import (
	"sync"
)

// Notification is an alias of struct{}, just for making the code easy to understand.
type Notification struct{}

// T represents tasks. Use New() to create a new  instance of T.
type T struct {
	// Error can hold an error that returned by the task worker.
	Error error

	startC    chan Notification
	doneC     chan Notification
	abortingC chan Notification
	started   bool
	done      bool
	canceled  bool
	aborting  bool
	mutex     sync.Mutex
}

// New returnes a new instance of T.
func New() *T {
	return &T{
		startC:    make(chan Notification, 1),
		doneC:     make(chan Notification, 1),
		abortingC: make(chan Notification, 1),
		started:   false,
		done:      false,
		canceled:  false,
		aborting:  false,
	}
}

// Start marks the task as under processing by a worker.
// Task workers must call this method before starting work on the task.
// When the return value is false, the task has been canceled, ignore the task in the case.
func (t *T) Start() (ok bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.canceled {
		t.started = true
		t.startC <- Notification{}
	}

	return !t.canceled
}

func (t *T) Do(f func()) {
	if t.Start() {
		f()
		t.Done()
	}
}

// IsAborting returns true if the task has been requested to abort.
// Task workers are supposed to stop working on the task once got true from this method.
func (t *T) IsAborting() bool {
	return t.aborting
}

// AbortingC returns a channel that will be notified when the task get a request to abort.
func (t *T) AbortingC() <-chan Notification {
	return t.abortingC
}

// Done notifies that the processing on the task has been completed.
// This method panics when the task has been canceled or not started yet.
func (t *T) Done() {
	if !t.started {
		panic("task must be Start()ed before Done()")
	}
	if t.canceled {
		panic("canceled task must not be Done()")
	}
	t.done = true
	t.doneC <- Notification{}
}

// Cancel requests the task not to be processed.
// Returns true when the task has been canceled successfully before it's started,
// false when it's already started and failed to cancel.
func (t *T) Cancel() (success bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.started {
		return false
	}

	t.canceled = true
	return true
}

// Abort requests the task runner to stop working on the task.
// This method is a non-blocking and returns immediately.
// This method panics when the task has been canceled.
func (t *T) Abort() {
	if t.canceled {
		panic("aborting a canceled task")
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.aborting = true
	t.abortingC <- Notification{}
}

// IsStarted returns true when the task is started.
// When the task has been canceled, this method panics.
func (t *T) IsStarted() bool {
	if t.canceled {
		panic("waiting for a canceled task to start")
	}
	return t.started
}

// WaitStart block until the task has started.
// This is a blocking method. Use WaitStartC() for non-blocking wait.
// When the task has been canceled, this method panics.
func (t *T) WaitStart() {
	if t.canceled {
		panic("waiting for a canceled task to start")
	}
	<-t.startC
}

// WaitStartC returns a channel to wait for the task to start.
// This method is non-blocking and returns immediately.
// When the task has been canceled, this method panics.
func (t *T) WaitStartC() <-chan Notification {
	if t.canceled {
		panic("waiting for a canceled task to start")
	}
	return t.startC
}

// IsDone returns true if the task has been completed.
// When the task has been canceled, this method panics.
func (t *T) IsDone() bool {
	if t.canceled {
		panic("waiting for a canceled task to done")
	}

	return t.done
}

// WaitDone blocks until the task has been Done().
// This is a blocking method. Use WaitC() for non-blocking wait.
// When the task has been canceled, this method panics.
func (t *T) WaitDone() {
	if t.canceled {
		panic("waiting for a canceled task to be done")
	}

	<-t.doneC
}

// WaitDoneC returns a channel to wait for the task to be done.
// This method is non blocking and returns immediately.
// When the task has been canceled, this method panics.
func (t *T) WaitDoneC() <-chan Notification {
	if t.canceled {
		panic("waiting for a canceled task to be done")
	}

	return t.doneC
}
