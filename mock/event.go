package mock

import (
	"context"

	"github.com/saiddis/todev"
)

var _ todev.EventService = (*EventService)(nil)

type EventService struct {
	PublishEventFn    func(id int, event todev.Event) error
	SubscribeFn       func(ctx context.Context) (todev.Subscription, error)
	GetSubscribtionFn func(id int) (todev.Subscription, bool)
}

func (s *EventService) PublishEvent(id int, event todev.Event) error {
	return s.PublishEventFn(id, event)
}

func (s *EventService) Subscribe(ctx context.Context) (todev.Subscription, error) {
	return s.SubscribeFn(ctx)
}

func (s *EventService) GetSubscribtion(id int) (todev.Subscription, bool) {
	return s.GetSubscribtionFn(id)
}

type Subscription struct {
	CloseFn func()
	CFn     func() <-chan todev.Event
	DoneFn  func() chan struct{}
}

func (s Subscription) Close() {
	s.CloseFn()
}

func (s Subscription) C() <-chan todev.Event {
	return s.CFn()
}

func (s Subscription) Done() chan struct{} {
	return s.DoneFn()
}
