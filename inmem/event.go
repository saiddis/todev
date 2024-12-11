package inmem

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/saiddis/todev"
)

// EventBufferSize is the buffer size of the channel for each subscription.
const EventBufferSize = 16

// EventService is the concrete implementation of the EventService interface.
type EventService struct {
	mu sync.RWMutex
	m  map[int]map[*Subscription]struct{} // subscriptions by user ID
}

// NewEventService creates and initializes a new EventService.
func NewEventService() *EventService {
	return &EventService{
		m: make(map[int]map[*Subscription]struct{}),
	}
}

// PublishEvent publishes an event to all subscriptions of a user.
// If a subscription's channel is full, it will be unsubscribed.
func (s *EventService) PublishEvent(userID int, event todev.Event) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subs, ok := s.m[userID]
	if !ok || len(subs) == 0 {
		return fmt.Errorf("no subscriptions found for user ID %d", userID)
	}

	for sub := range subs {
		select {
		case sub.eventChan <- event:
		default:
			s.unsubscribe(sub)
		}
	}
	return nil
}

// Subscribe creates a new subscription for the current user's events.
func (s *EventService) Subscribe(ctx context.Context) (todev.Subscription, error) {
	userID, ok := ctx.Value("userID").(int)
	if !ok || userID == 0 {
		return nil, errors.New("user ID is missing or invalid in context")
	}

	sub := &Subscription{
		service:   s,
		userID:    userID,
		eventChan: make(chan todev.Event, EventBufferSize),
		doneChan:  make(chan struct{}),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	subs, ok := s.m[userID]
	if !ok {
		subs = make(map[*Subscription]struct{})
		s.m[userID] = subs
	}
	subs[sub] = struct{}{}

	return sub, nil
}

// GetSubscribtion returns the subscriptions for a user if it exists.
func (s *EventService) GetSubscribtion(userID int) (todev.Subscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subs, ok := s.m[userID]
	if !ok || len(subs) == 0 {
		return nil, false
	}

	// Return the first subscription (arbitrary choice since we allow multiple).
	for sub := range subs {
		return sub, true
	}
	return nil, false
}

func (s *EventService) unsubscribe(sub *Subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find subscription map for the user.
	subs, ok := s.m[sub.userID]
	if !ok {
		return
	}

	delete(subs, sub)

	// Remove the user from the map if no subscriptions remain.
	if len(subs) == 0 {
		delete(s.m, sub.userID)
	}

	// Ensure the subscription is cleaned up.
	sub.once.Do(func() {
		close(sub.eventChan)
		close(sub.doneChan)
	})
}

// SubscriptionImpl is the concrete implementation of the Subscription interface.
type Subscription struct {
	service   *EventService
	userID    int
	eventChan chan todev.Event
	doneChan  chan struct{}
	once      sync.Once
}

// C returns the channel for receiving events.
func (s *Subscription) C() <-chan todev.Event {
	return s.eventChan
}

// Done returns the channel used for unsubscribing.
func (s *Subscription) Done() chan struct{} {
	return s.doneChan
}

// Close unsubscribes the subscription from the service and cleans up resources.
func (s *Subscription) Close() {
	s.service.unsubscribe(s)
}
