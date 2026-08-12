package postgresidempotency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/waittest"
)

func TestPublicationGroup(t *testing.T) {
	var identity [32]byte
	identity[0] = 1
	done := make(chan struct{})
	group := publicationGroup{waiting: map[[32]byte]chan struct{}{identity: done}}
	followerCalled := make(chan struct{})
	followerDone := make(chan struct {
		leader bool
		err    error
	}, 1)
	go func() {
		leader, err := group.run(t.Context(), identity, func() error {
			close(followerCalled)
			return errors.New("follower entered publication callback")
		})
		followerDone <- struct {
			leader bool
			err    error
		}{leader: leader, err: err}
	}()
	close(done)
	follower := waittest.Receive(t, followerDone, time.Second, "publication follower")
	if follower.err != nil || follower.leader {
		t.Fatalf("follower = (%t, %v), want (false, nil)", follower.leader, follower.err)
	}
	select {
	case <-followerCalled:
		t.Fatal("follower entered publication callback")
	default:
	}

	group = publicationGroup{waiting: make(map[[32]byte]chan struct{})}
	leader, err := group.run(t.Context(), identity, func() error { return nil })
	if err != nil || !leader {
		t.Fatalf("leader publication = (%t, %v), want (true, nil)", leader, err)
	}
	if len(group.waiting) != 0 {
		t.Fatalf("completed leader left %d publication signals", len(group.waiting))
	}
}

func TestPublicationGroupFollowerCancellation(t *testing.T) {
	var identity [32]byte
	identity[0] = 2
	group := publicationGroup{waiting: map[[32]byte]chan struct{}{identity: make(chan struct{})}}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := group.run(ctx, identity, func() error {
		return errors.New("canceled follower entered publication callback")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("follower error = %v, want context canceled", err)
	}
}
