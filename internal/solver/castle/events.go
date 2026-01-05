package castle

import (
	"container/heap"
	"sync/atomic"
)

// EventType represents the type of simulation event
type EventType int

const (
	EventMissionComplete EventType = iota
	EventBuildingComplete
	EventResearchComplete
	EventTrainingComplete
	EventStateChanged // Dummy event to re-evaluate decisions
)

// String returns a string representation of the event type
func (et EventType) String() string {
	switch et {
	case EventMissionComplete:
		return "MissionComplete"
	case EventBuildingComplete:
		return "BuildingComplete"
	case EventResearchComplete:
		return "ResearchComplete"
	case EventTrainingComplete:
		return "TrainingComplete"
	case EventStateChanged:
		return "StateChanged"
	default:
		return "Unknown"
	}
}

// Priority returns the processing priority for this event type
// Lower priority = processed first when events have same time
func (et EventType) Priority() int {
	switch et {
	case EventMissionComplete:
		return 0 // First: add resources, return units
	case EventBuildingComplete:
		return 1 // Second: update production/storage
	case EventResearchComplete:
		return 2 // Third: unlock techs/units
	case EventTrainingComplete:
		return 3 // Fourth: add unit to army
	case EventStateChanged:
		return 10 // Last: make decisions with updated state
	default:
		return 99
	}
}

// Event represents a simulation event
type Event struct {
	Time     int // Seconds from simulation start
	Type     EventType
	Payload  any   // BuildingAction, ResearchAction, TrainUnitAction, MissionState
	Sequence int64 // Global insertion order for stable sorting
}

// Global sequence counter for deterministic event ordering
var eventSequence int64

// eventHeap implements heap.Interface for min-heap of Events
type eventHeap []Event

func (h eventHeap) Len() int { return len(h) }

func (h eventHeap) Less(i, j int) bool {
	// Sort by Time first
	if h[i].Time != h[j].Time {
		return h[i].Time < h[j].Time
	}
	// Then by Priority
	if h[i].Type.Priority() != h[j].Type.Priority() {
		return h[i].Type.Priority() < h[j].Type.Priority()
	}
	// Finally by Sequence for stability (insertion order)
	return h[i].Sequence < h[j].Sequence
}

func (h eventHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *eventHeap) Push(x any) {
	*h = append(*h, x.(Event))
}

func (h *eventHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// EventQueue is a priority queue for events using a min-heap
// Events are sorted by (Time, Priority, Sequence) for deterministic ordering
type EventQueue struct {
	h eventHeap
}

// NewEventQueue creates a new empty event queue
func NewEventQueue() *EventQueue {
	q := &EventQueue{
		h: make(eventHeap, 0),
	}
	heap.Init(&q.h)
	return q
}

// Push adds an event to the queue with automatic sequence assignment
func (q *EventQueue) Push(e Event) {
	// Assign sequence number for stable ordering
	e.Sequence = atomic.AddInt64(&eventSequence, 1)
	heap.Push(&q.h, e)
}

// Pop removes and returns the minimum event
func (q *EventQueue) Pop() Event {
	if len(q.h) == 0 {
		return Event{Time: -1}
	}
	return heap.Pop(&q.h).(Event)
}

// Peek returns the minimum event without removing it
func (q *EventQueue) Peek() Event {
	if len(q.h) == 0 {
		return Event{Time: -1}
	}
	return q.h[0]
}

// Empty returns true if the queue has no events
func (q *EventQueue) Empty() bool {
	return len(q.h) == 0
}

// Len returns the number of events in the queue
func (q *EventQueue) Len() int {
	return len(q.h)
}

// PushIfNotExists adds a StateChanged event only if one doesn't already exist at that time
// This prevents duplicate re-evaluations
func (q *EventQueue) PushIfNotExists(e Event) {
	if e.Type != EventStateChanged {
		q.Push(e)
		return
	}

	// Check if StateChanged already exists at this time
	for i := 0; i < len(q.h); i++ {
		if q.h[i].Time == e.Time && q.h[i].Type == EventStateChanged {
			return // Already exists
		}
	}

	q.Push(e)
}

// Clear removes all events from the queue
func (q *EventQueue) Clear() {
	q.h = q.h[:0]
}

// Events returns a copy of all events in heap order (for debugging)
func (q *EventQueue) Events() []Event {
	result := make([]Event, len(q.h))
	copy(result, q.h)
	return result
}
