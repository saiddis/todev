package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/saiddis/todev"
)

type EventService struct {
	Repos map[int]Subscribtion
}

func (s *EventService) PublishEvent(repoID int, event todev.Event) error {
	if sub, ok := s.Repos[repoID]; ok {
		sub.Event <- event
		return nil
	}
	return fmt.Errorf("no repos with id: %d", repoID)
}

func (s *EventService) Subscribe(ctx context.Context) (todev.Subscription, error) {
	sub := Subscribtion{
		Event: make(chan todev.Event),
		done:  make(chan struct{}),
	}

	if repoID, ok := ctx.Value("repoID").(int); ok {
		s.Repos[repoID] = sub
		return sub, nil
	}

	return Subscribtion{}, errors.New("failed to convert value from context")
}

func (s *EventService) GetSubscribtion(repoID int) (todev.Subscription, bool) {
	sub, ok := s.Repos[repoID]
	return sub, ok
}

type Subscribtion struct {
	Event chan todev.Event
	done  chan struct{}
}

func NewSubscribtion() *Subscribtion {
	return &Subscribtion{}
}

func (s Subscribtion) C() <-chan todev.Event {
	return s.Event
}

func (s Subscribtion) Done() chan struct{} {
	return s.done
}

func (s Subscribtion) Close() {
	close(s.Event)
	close(s.done)
}
