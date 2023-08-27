package eventbus

import (
	"context"
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus/queue"
	"github.com/rprtr258/pm/internal/infra/db"
)

const _tickInterval = 500 * time.Millisecond

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

func (e Event) String() string {
	return fmt.Sprintf("%s:%v", e.Kind, e.Data)
}

type EmitReason int

const (
	EmitReasonDied EmitReason = iota
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
	PID        int
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
	q  *queue.Queue[Event]
	db db.Handle

	mu          sync.Mutex
	subscribers map[string]Subscriber
}

func New(db db.Handle) *EventBus {
	return &EventBus{
		q:           queue.New[Event](),
		db:          db,
		mu:          sync.Mutex{},
		subscribers: map[string]Subscriber{},
	}
}

func (e *EventBus) Start(ctx context.Context) {
	defer func() {
		for _, sub := range e.subscribers {
			close(sub.Chan)
		}
	}()

	tick := time.NewTicker(_tickInterval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			event, ok := e.q.Pop()
			if !ok {
				continue
			}

			log.Debug().
				Stringer("event", event).
				Msg("got event, routing")

			e.mu.Lock()
			for name, sub := range e.subscribers {
				if _, ok := sub.Kinds[event.Kind]; !ok {
					continue
				}

				log.Debug().
					Any("event", event).
					Str("subscriber", name).
					Msg("publishing event")
				// NOTE: blocks on every subscriber
				select {
				case sub.Chan <- event:
				case <-ctx.Done():
					return
				}
			}
			e.mu.Unlock()
		}
	}
}

func (e *EventBus) Publish(ctx context.Context, events ...Event) {
	for _, event := range events {
		select {
		case <-ctx.Done():
			return
		default:
			if event.Kind == KindProcStopRequest {
				data := event.Data.(DataProcStopRequest) //nolint:errcheck,forcetypeassert // not needed
				procID := data.ProcID
				proc, ok := e.db.GetProc(procID)
				if !ok {
					log.Error().
						Uint64("proc_id", procID).
						Msg("proc not found")
					return
				}

				if proc.Status.Status != core.StatusRunning {
					log.Error().
						Uint64("proc_id", procID).
						Msg("proc is not running")
					return
				}

				data.PID = proc.Status.Pid
				event.Data = data
			}

			e.q.Push(event)
		}
	}
}

func NewPublishProcStarted(proc core.Proc, pid int, emitReason EmitReason) Event {
	if emitReason&(emitReason-1) != 0 {
		log.Warn().
			Stringer("reason", emitReason).
			Msg("invalid emit reason for proc started event")
		// return
	}

	return Event{
		Kind: KindProcStarted,
		Data: DataProcStarted{
			Proc:       proc,
			Pid:        pid,
			At:         time.Now(),
			EmitReason: emitReason,
		},
	}
}

func NewPublishProcStopped(procID core.ProcID, exitCode int, emitReason EmitReason) Event {
	if emitReason&(emitReason-1) != 0 {
		log.Warn().
			Stringer("reason", emitReason).
			Msg("invalid emit reason for proc stopped event")
		// return
	}

	return Event{
		Kind: KindProcStopped,
		Data: DataProcStopped{
			ProcID:     procID,
			ExitCode:   exitCode,
			At:         time.Now(),
			EmitReason: emitReason,
		},
	}
}

func NewPublishProcStartRequest(procID core.ProcID, emitReason EmitReason) Event {
	log.Debug().
		Uint64("proc_id", procID).
		Stringer("emit_reason", emitReason).
		Msg("publishing proc start request")
	return Event{
		Kind: KindProcStartRequest,
		Data: DataProcStartRequest{
			ProcID:     procID,
			EmitReason: emitReason,
		},
	}
}

func NewPublishProcStopRequest(procID core.ProcID, emitReason EmitReason) Event {
	return Event{
		Kind: KindProcStopRequest,
		Data: DataProcStopRequest{
			ProcID:     procID,
			PID:        0, // NOTE: filled on Publish
			EmitReason: emitReason,
		},
	}
}

func NewPublishProcSignalRequest(signal syscall.Signal, procIDs ...core.ProcID) Event {
	return Event{
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
