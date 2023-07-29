package eventbus

import (
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/rprtr258/pm/internal/core"
	"golang.org/x/exp/slog"
)

type EventKind string

const (
	KindProcStarted       EventKind = "ProcStarted"
	KindProcStopped       EventKind = "ProcStopped"
	KindProcStartRequest  EventKind = "ProcStartRequest"
	KindProcStopRequest   EventKind = "ProcStopRequest"
	KindProcSignalRequest EventKind = "ProcSignalRequest"
)

func (e EventKind) String() string {
	switch e {
	case KindProcStarted, KindProcStopped,
		KindProcStartRequest, KindProcStopRequest, KindProcSignalRequest:
		return string(e)
	default:
		return fmt.Sprintf("Unknown:%s", string(e))
	}
}

type Event struct {
	Kind EventKind
	Data any
}

type EmitReason int

const (
	EmitReasonDied EmitReason = 1 << iota
	EmitReasonByUser
	EmitReasonByWatcher
)

func (e EmitReason) String() string {
	switch e {
	case EmitReasonDied:
		return "Died"
	case EmitReasonByUser:
		return "ByUser"
	case EmitReasonByWatcher:
		return "ByWatcher"
	default:
		return fmt.Sprintf("Unknown:%d", e)
	}
}

type DataProcStarted struct {
	Proc core.Proc
	At   time.Time
	Pid  int

	// EmitReason = ByUser | ByWatcher
	EmitReason EmitReason
}

type DataProcStopped struct {
	ProcID   core.ProcID
	ExitCode int
	At       time.Time

	// EmitReason = Died | ByUser | ByWatcher
	EmitReason EmitReason
}

type DataProcStartRequest struct {
	ProcID     core.ProcID
	EmitReason EmitReason
}

type DataProcStopRequest struct {
	ProcID     core.ProcID
	EmitReason EmitReason
}

type DataProcSignalRequest struct {
	ProcIDs []core.ProcID
	Signal  syscall.Signal
}

type Subscriber struct {
	Kinds map[EventKind]struct{}
	Chan  chan Event
}

type EventBus struct {
	eventsCh chan Event
	doneCh   chan struct{}

	mu          sync.Mutex
	subscribers map[string]Subscriber
}

func New() *EventBus {
	return &EventBus{
		doneCh:      make(chan struct{}),
		eventsCh:    make(chan Event),
		mu:          sync.Mutex{},
		subscribers: map[string]Subscriber{},
	}
}

func (e *EventBus) Start() {
	go func() {
		for {
			select {
			case <-e.doneCh:
				return
			case event := <-e.eventsCh:
				slog.Debug(
					"got event, routing",
					slog.Any("event", event),
				)

				e.mu.Lock()
				for name, sub := range e.subscribers {
					slog.Debug(
						"publishing event",
						slog.Any("event", event),
						slog.String("subscriber", name),
					)
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

func (e *EventBus) PublishProcStarted(proc core.Proc, pid int, emitReason EmitReason) {
	if emitReason&(emitReason-1) != 0 {
		slog.Warn(
			"invalid emit reason for proc started event",
			slog.String("reason", emitReason.String()),
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

func (e *EventBus) PublishProcStopped(procID core.ProcID, exitCode int, emitReason EmitReason) {
	if emitReason&(emitReason-1) != 0 {
		slog.Warn(
			"invalid emit reason for proc stopped event",
			slog.String("reason", emitReason.String()),
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

func (e *EventBus) PublishProcStartRequest(procID core.ProcID, emitReason EmitReason) {
	slog.Debug(
		"publishing proc start request",
		slog.Uint64("proc_id", procID),
		slog.String("emit_reason", emitReason.String()),
	)
	e.eventsCh <- Event{
		Kind: KindProcStartRequest,
		Data: DataProcStartRequest{
			ProcID:     procID,
			EmitReason: emitReason,
		},
	}
}

func (e *EventBus) PublishProcStopRequest(procID core.ProcID, emitReason EmitReason) {
	e.eventsCh <- Event{
		Kind: KindProcStopRequest,
		Data: DataProcStopRequest{
			ProcID:     procID,
			EmitReason: emitReason,
		},
	}
}

func (e *EventBus) PublishProcSignalRequest(signal syscall.Signal, procIDs ...core.ProcID) {
	e.eventsCh <- Event{
		Kind: KindProcSignalRequest,
		Data: DataProcSignalRequest{
			ProcIDs: procIDs,
			Signal:  signal,
		},
	}
}

func (e *EventBus) Subscribe(name string, kinds ...EventKind) <-chan Event {
	kindsSet := make(map[EventKind]struct{}, len(kinds))
	for _, kind := range kinds {
		kindsSet[kind] = struct{}{}
	}

	e.mu.Lock()
	if _, ok := e.subscribers[name]; ok {
		panic(fmt.Sprintf("duplicate subscriber: %s", name))
	}

	ch := make(chan Event)
	e.subscribers[name] = Subscriber{
		Kinds: kindsSet,
		Chan:  ch,
	}
	e.mu.Unlock()

	return ch
}
