package eventbus

import (
	"context"
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/rprtr258/fun/set"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
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

	// EmitReason = ByUser | ByWatcher
	EmitReason EmitReason
}

type DataProcStopped struct {
	ProcID core.PMID
	At     time.Time

	// EmitReason = Died | ByUser | ByWatcher
	EmitReason EmitReason
}

type DataProcStartRequest struct {
	ProcID     core.PMID
	EmitReason EmitReason
}

type DataProcStopRequest struct {
	ProcID     core.PMID
	EmitReason EmitReason
}

type DataProcSignalRequest struct {
	ProcID core.PMID
	Signal syscall.Signal
}

type Subscriber struct {
	Kinds set.Set[EventKind]
	Queue chan Event
}

type EventBus struct {
	q  chan Event
	db db.Handle

	mu          sync.Mutex
	subscribers map[string]Subscriber
}

func Module(db db.Handle) *EventBus {
	return &EventBus{
		q:           make(chan Event, 100),
		db:          db,
		mu:          sync.Mutex{},
		subscribers: map[string]Subscriber{},
	}
}

func (e *EventBus) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-e.q:
			log.Debug().
				Stringer("event", event).
				Msg("got event, routing")

			func() {
				e.mu.Lock()
				defer e.mu.Unlock()

				for name, sub := range e.subscribers {
					select {
					case <-ctx.Done():
						return
					default:
					}

					if !sub.Kinds.Contains(event.Kind) {
						continue
					}

					log.Debug().
						Any("event", event).
						Str("subscriber", name).
						Msg("publishing event")
					sub.Queue <- event
				}
			}()
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
						Stringer("pmid", procID).
						Msg("proc not found")
					continue
				}

				if proc.Status.Status != core.StatusRunning {
					log.Error().
						Stringer("pmid", procID).
						Msg("proc is not running")
					continue
				}

				data.ProcID = proc.ID
				event.Data = data
			}

			e.q <- event
		}
	}
}

func NewPublishProcStarted(proc core.Proc, emitReason EmitReason) Event {
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
			At:         time.Now(),
			EmitReason: emitReason,
		},
	}
}

func NewPublishProcStopped(procID core.PMID, emitReason EmitReason) Event {
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
			At:         time.Now(),
			EmitReason: emitReason,
		},
	}
}

func NewPublishProcStartRequest(procID core.PMID, emitReason EmitReason) Event {
	log.Debug().
		Stringer("pmid", procID).
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

func NewPublishProcStopRequest(procID core.PMID, emitReason EmitReason) Event {
	return Event{
		Kind: KindProcStopRequest,
		Data: DataProcStopRequest{
			ProcID:     procID,
			EmitReason: emitReason,
		},
	}
}

func NewPublishProcSignalRequest(signal syscall.Signal, procID core.PMID) Event {
	return Event{
		Kind: KindProcSignalRequest,
		Data: DataProcSignalRequest{
			ProcID: procID,
			Signal: signal,
		},
	}
}

func (e *EventBus) Subscribe(name string, kinds ...EventKind) <-chan Event {
	kindsSet := set.New[EventKind](len(kinds))
	kindsSet.Add(kinds...)

	e.mu.Lock()
	if _, ok := e.subscribers[name]; ok {
		panic(fmt.Sprintf("duplicate subscriber: %s", name))
	}

	q := make(chan Event, 10)
	e.subscribers[name] = Subscriber{
		Kinds: kindsSet,
		Queue: q,
	}
	e.mu.Unlock()

	return q
}
