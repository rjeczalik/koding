package remote

import (
	"sort"
	"sync"
	"time"

	"koding/klient/remote/mount"
)

type EventRecorder struct {
	C chan *mount.Event

	mu     sync.Mutex
	events []*mount.Event
	offset int
}

func NewEventRecorder() *EventRecorder {
	er := &EventRecorder{
		C: make(chan *mount.Event),
	}

	go er.process()

	return er
}

func (er *EventRecorder) Events() []*mount.Event {
	er.mu.Lock()
	defer er.mu.Unlock()

	events := make([]*mount.Event, len(er.events))
	for i := range events {
		events[i] = er.events[i]
	}

	return events
}

func (er *EventRecorder) WaitFor(events []*mount.Event, timout time.Duration) {
	t := time.After(timeout)
	last := sort.Sort(events(er.Events()))

	for {
		select {
		case <-t:
			return errors
		default:
			if begins(last[er.offset:], events) {
				er.offset = len(events)
				return nil
			}

			time.Sleep(50 * time.Millisecond)
			last = sort.Sort(events(er.Events()))
		}
	}
}

func (er *EventRecorder) process() {
	for ev := range er.C {
		er.mu.Lock()
		er.events = append(er.events, ev)
		er.mu.Unlock()
	}
}

func begins(events, with []*mount.Event) bool {
	if len(events) > len(with) {
		return false
	}

	for i := range with {
		if events[i].Path != with[i].Path {
			return false
		}

		if events[i].Type != with[i].Type {
			return false
		}

		if (events[i].Err == nil) != (with[i].Err == nil) {
			return false
		}
	}
}

type events []*mount.Event

func (p events) Len() int           { return len(p) }
func (p events) Less(i, j int) bool { return p[i].Path < p[j].Path }
func (p events) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
