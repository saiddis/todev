package inmem_test

import (
	"context"
	"testing"

	"github.com/saiddis/todev"
	"github.com/saiddis/todev/inmem"
)

func TestEventService(t *testing.T) {
	t.Run("Subscribe", func(t *testing.T) {
		ctx := context.Background()
		ctx0 := todev.NewContextWithUser(ctx, &todev.User{ID: 1})
		ctx1 := todev.NewContextWithUser(ctx, &todev.User{ID: 2})

		s := inmem.NewEventService()
		sub0a, err := s.Subscribe(ctx0)
		if err != nil {
			t.Fatal(err)
		}

		sub0b, err := s.Subscribe(ctx0)
		if err != nil {
			t.Fatal(err)
		}

		sub1, err := s.Subscribe(ctx1)
		if err != nil {
			t.Fatal(err)
		}

		s.PublishEvent(1, todev.Event{Type: "test1"})

		select {
		case <-sub0a.C():
		default:
			t.Fatalf("expected event")
		}

		select {
		case <-sub0b.C():
		default:
			t.Fatalf("expected event")
		}

		// Ensure second user does not recieve event.
		select {
		case <-sub1.C():
			t.Fatalf("expected no event")
		default:
		}

	})

	t.Run("Unsubscribe", func(t *testing.T) {
		ctx := context.Background()
		ctx0 := todev.NewContextWithUser(ctx, &todev.User{ID: 1})

		s := inmem.NewEventService()
		sub, err := s.Subscribe(ctx0)
		if err != nil {
			t.Fatal(err)
		}

		s.PublishEvent(1, todev.Event{Type: "test1"})
		sub.Close()

		select {
		case <-sub.C():
		default:
			t.Fatal("expected event")
		}

		// Ensure channel is now closed.
		if _, ok := <-sub.C(); ok {
			t.Fatalf("expected closed channel")
		}

		// Ensure unsubscribing twice is ok.
		sub.Close()
	})
}
