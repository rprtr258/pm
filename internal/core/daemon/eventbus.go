package daemon

import "github.com/rprtr258/pm/internal/core"

type EventKind int

const (
	EventKindProcStarted EventKind = iota
	EventKindProcStopped
)

type Event struct {
	Kind EventKind
	Data any
}

type EventDataProcStarted struct {
	ID core.ProcID
}

type EventDataProcStopped struct {
	ID core.ProcID
}

type Subscriber struct {
	Kinds map[EventKind]struct{}
	Chan  chan Event
}

type EventBus struct {
	Subscribers []Subscriber
	eventsCh    chan Event
	doneCh      chan struct{}
}

func NewEventBus() EventBus {
	return EventBus{
		Subscribers: nil,
		doneCh:      make(chan struct{}),
		eventsCh:    make(chan Event),
	}
}

func (e *EventBus) Start() {
	go func() {
		for {
			select {
			case <-e.doneCh:
			case event := <-e.eventsCh:
				for _, sub := range e.Subscribers {
					// NOTE: blocks on every subscriber
					sub.Chan <- event
				}
			}
		}
	}()
}

func (e *EventBus) Close() {
	close(e.doneCh)
	close(e.eventsCh)
	for _, sub := range e.Subscribers {
		close(sub.Chan)
	}
}

func (e *EventBus) Publish(event Event) {
	e.eventsCh <- event
}

func (e *EventBus) Subscribe(kinds ...EventKind) <-chan Event {
	kindsSet := make(map[EventKind]struct{}, len(kinds))
	for _, kind := range kinds {
		kindsSet[kind] = struct{}{}
	}

	ch := make(chan Event)
	e.Subscribers = append(e.Subscribers, Subscriber{
		Kinds: kindsSet,
		Chan:  ch,
	})

	return ch
}
