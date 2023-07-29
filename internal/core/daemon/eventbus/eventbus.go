package eventbus

import (
	"sync"
	"time"

	"github.com/rprtr258/pm/internal/core"
	"golang.org/x/exp/slog"
)

type EventKind int

const (
	KindProcStarted EventKind = iota
	KindProcStopped
	KindProcRestartRequest
)

type Event struct {
	Kind EventKind
	Data any
}

const (
	EmitReasonDied = 1 << iota
	EmitReasonByUser
	EmitReasonByWatcher
)

type DataProcStarted struct {
	Proc core.Proc
	At   time.Time
	Pid  int

	// EmitReason = ByUser | ByWatcher
	EmitReason int
}

type DataProcStopped struct {
	ProcID   core.ProcID
	ExitCode int
	At       time.Time

	// EmitReason = Died | ByUser | ByWatcher
	EmitReason int
}

type DataProcRestartRequest struct {
	ProcID core.ProcID
}

type Subscriber struct {
	Kinds map[EventKind]struct{}
	Chan  chan Event
}

type EventBus struct {
	eventsCh chan Event
	doneCh   chan struct{}

	mu          sync.Mutex
	subscribers []Subscriber
}

func New() *EventBus {
	return &EventBus{
		doneCh:      make(chan struct{}),
		eventsCh:    make(chan Event),
		mu:          sync.Mutex{},
		subscribers: nil,
	}
}

func (e *EventBus) Start() {
	go func() {
		for {
			select {
			case <-e.doneCh:
				return
			case event := <-e.eventsCh:
				e.mu.Lock()
				for _, sub := range e.subscribers {
					// NOTE: blocks on every subscriber
					select {
					case sub.Chan <- event:
					case <-e.doneCh:
						return
					}
				}
				e.mu.Unlock()
			}
		}
	}()
}

func (e *EventBus) Close() {
	close(e.doneCh)
	close(e.eventsCh)
	for _, sub := range e.subscribers {
		close(sub.Chan)
	}
}

func (e *EventBus) PublishProcStarted(proc core.Proc, pid int, emitReason int) {
	if emitReason&(emitReason-1) == 0 {
		slog.Warn(
			"invalid emit reason for proc started event",
			slog.Int("reason", emitReason),
		)
		return
	}

	e.eventsCh <- Event{
		Kind: KindProcStarted,
		Data: DataProcStarted{
			Proc:       proc,
			Pid:        pid,
			At:         time.Now(),
			EmitReason: emitReason,
		},
	}
}

func (e *EventBus) PublishProcStopped(procID core.ProcID, exitCode int, emitReason int) {
	if emitReason&(emitReason-1) == 0 {
		slog.Warn(
			"invalid emit reason for proc stopped event",
			slog.Int("reason", emitReason),
		)
		return
	}

	e.eventsCh <- Event{
		Kind: KindProcStopped,
		Data: DataProcStopped{
			ProcID:     procID,
			ExitCode:   exitCode,
			At:         time.Now(),
			EmitReason: emitReason,
		},
	}
}

func (e *EventBus) PublishProcRestartRequest(procID core.ProcID) {
	e.eventsCh <- Event{
		Kind: KindProcRestartRequest,
		Data: DataProcRestartRequest{
			ProcID: procID,
		},
	}
}

func (e *EventBus) Subscribe(kinds ...EventKind) <-chan Event {
	kindsSet := make(map[EventKind]struct{}, len(kinds))
	for _, kind := range kinds {
		kindsSet[kind] = struct{}{}
	}

	ch := make(chan Event)

	e.mu.Lock()
	e.subscribers = append(e.subscribers, Subscriber{
		Kinds: kindsSet,
		Chan:  ch,
	})
	e.mu.Unlock()

	return ch
}
