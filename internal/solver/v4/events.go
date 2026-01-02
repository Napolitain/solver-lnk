package v4

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
	Time    int       // Seconds from simulation start
	Type    EventType
	Payload any       // BuildingAction, ResearchAction, TrainUnitAction, MissionState
}

// EventQueue is a priority queue for events
// Events are sorted by (Time, Priority)
type EventQueue struct {
	events []Event
}

// NewEventQueue creates a new empty event queue
func NewEventQueue() *EventQueue {
	return &EventQueue{
		events: make([]Event, 0),
	}
}

// Push adds an event to the queue in sorted order
func (q *EventQueue) Push(e Event) {
	// Find insertion point (binary search would be faster for large queues)
	insertIdx := len(q.events)
	for i, existing := range q.events {
		if e.Time < existing.Time || (e.Time == existing.Time && e.Type.Priority() < existing.Type.Priority()) {
			insertIdx = i
			break
		}
	}

	// Insert at position
	q.events = append(q.events, Event{})
	copy(q.events[insertIdx+1:], q.events[insertIdx:])
	q.events[insertIdx] = e
}

// Pop removes and returns the first event
func (q *EventQueue) Pop() Event {
	if len(q.events) == 0 {
		return Event{Time: -1}
	}
	e := q.events[0]
	q.events = q.events[1:]
	return e
}

// Peek returns the first event without removing it
func (q *EventQueue) Peek() Event {
	if len(q.events) == 0 {
		return Event{Time: -1}
	}
	return q.events[0]
}

// Empty returns true if the queue has no events
func (q *EventQueue) Empty() bool {
	return len(q.events) == 0
}

// Len returns the number of events in the queue
func (q *EventQueue) Len() int {
	return len(q.events)
}

// PushIfNotExists adds a StateChanged event only if one doesn't already exist at that time
// This prevents duplicate re-evaluations
func (q *EventQueue) PushIfNotExists(e Event) {
	if e.Type != EventStateChanged {
		q.Push(e)
		return
	}

	// Check if StateChanged already exists at this time
	for _, existing := range q.events {
		if existing.Time == e.Time && existing.Type == EventStateChanged {
			return // Already exists
		}
	}

	q.Push(e)
}

// Clear removes all events from the queue
func (q *EventQueue) Clear() {
	q.events = q.events[:0]
}

// Events returns a copy of all events (for debugging)
func (q *EventQueue) Events() []Event {
	result := make([]Event, len(q.events))
	copy(result, q.events)
	return result
}
