package event

type Event struct {
	trigger chan struct{}
}

func New() Event {
	return Event {
		trigger: make(chan struct{}),
	}
}

func (event *Event) IsSet() bool {
	select {
	case <-event.trigger: return true
	default: return false
	}
}

func (event *Event) Set() {
	close(event.trigger)
}

func (event *Event) Wait() {
	<-event.trigger
}

func (event *Event) Select() <-chan struct{} {
	return event.trigger
}
