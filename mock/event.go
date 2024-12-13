package mock

import (
	"context"

	"github.com/saiddis/todev"
)

var _ todev.EventService = (*EventService)(nil)

type EventService struct {
	PublishEventFn    func(id int, event todev.Event)
	SubscribeFn       func(ctx context.Context) (todev.Subscription, error)
	GetSubscribtionFn func(id int) (todev.Subscription, bool)
}

func (s *EventService) PublishEvent(id int, event todev.Event) {
	s.PublishEventFn(id, event)
}

func (s *EventService) Subscribe(ctx context.Context) (todev.Subscription, error) {
	return s.SubscribeFn(ctx)
}

func (s *EventService) GetSubscription(id int) (todev.Subscription, bool) {
	return s.GetSubscribtionFn(id)
}

type Subscription struct {
	CloseFn func()
	CFn     func() <-chan todev.Event
}

func (s Subscription) Close() {
	s.CloseFn()
}

func (s Subscription) C() <-chan todev.Event {
	return s.CFn()
}
