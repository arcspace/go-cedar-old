package process

import (
	"context"
	"time"

	"github.com/arcspace/go-cedar/log"
)

// NilContext is used to start Contexts with no parent Context.
var NilContext = Context((*ctx)(nil))

// Start starts the given context as its own process root.
func Start(task *Task) (Context, error) {
	return NilContext.StartChild(task)
}

func Go(parent Context, label string, fn func(ctx Context)) (Context, error) {
	return parent.StartChild(&Task{
		Label: label,
		OnRun: fn,
	})
}

// Task is an optional set of callbacks for a Context
type Task struct {
	Label     string
	IdleClose time.Duration           // If > 0, CloseWhenIdle() is automatically called after the last remaining child is closed or after OnRun() completes (if set), whichever occurs later.
	Ref       interface{}             // Offered to you to store anything and accessed via Context.TaskRef()
	OnStart   func(ctx Context) error // Blocking fn called in StartChild(). If err, ctx.Close() is called and Go() returns the err and OnRun is never called.
	OnRun     func(ctx Context)       // Async work body. If non-nil, ctx.Close() will be automatically called after OnRun() completes
	OnClosing func()                  // Called after Close() is first called and immediately before children are signaled to close.
	OnClosed  func()                  // Called after Close() and all children have completed Close() (but immediately before Done() is released)
}

type Context interface {
	log.Logger

	// A process.Context can be used just like a context.Context.
	context.Context

	// Returns Task.Ref passed to StartChild()
	TaskRef() interface{}

	// The context's public label
	ContextLabel() string

	// A guaranteed unique ID assigned after Start() is called.
	ContextID() int64

	// Creates a new child Context with for given Task.
	// If OnStart() returns an error error is encountered, then child.Close() is immediately called and the error is returned.
	StartChild(task *Task) (Context, error)

	// Convenience function for StartChild() and is equivalent to:
	//
	//      parent.StartChild(label, &Task{
	//  		IdleClose: time.Nanosecond,
	// 	        OnRun: fn,
	//      })
	Go(label string, fn func(ctx Context)) (Context, error)

	// Appends all currently open/active child Contexts to the given slice and returns the given slice.
	// Naturally, the returned items are back-ward looking as any could close at any time.
	// Context implementations wishing to remain lightweight may opt to not retain a list of children (and just return the given slice as-is).
	GetChildren(in []Context) []Context

	// Async call that initiates process shutdown and causes all children's Close() to be called.
	// Close can be called multiple times but calls after the first are in effect ignored.
	// First, child processes get Close() in breath-first order.
	// After all children are done closing, OnClosing(), then OnClosed() are executed.
	Close() error

	// Inserts a pending Close() on this Context once it is idle after the given delay.
	// Subsequent calls will update the delay but the previously pending delay must run out first.
	// If at the end of the period Task.OnRun() is complete and there are no children, then Close() is called.
	CloseWhenIdle(delay time.Duration)

	// Signals when Close() has been called.
	// First, Child processes get Close(),  then OnClosing, then OnClosed are executing
	Closing() <-chan struct{}

	// Signals when Close() has fully executed, no children remain, and OnClosed() has been completed.
	Done() <-chan struct{}
}
