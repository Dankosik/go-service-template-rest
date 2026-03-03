package health

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestServiceReadySuccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	db := NewMockProbe(ctrl)
	db.EXPECT().Check(gomock.Any()).Return(nil)

	cache := NewMockProbe(ctrl)
	cache.EXPECT().Check(gomock.Any()).Return(nil)

	svc := New(db, cache)

	if err := svc.Ready(context.Background()); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
}

func TestServiceReadyFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	downErr := errors.New("down")

	db := NewMockProbe(ctrl)
	db.EXPECT().Name().Return("db")
	db.EXPECT().Check(gomock.Any()).Return(downErr)

	svc := New(db)

	err := svc.Ready(context.Background())
	if err == nil {
		t.Fatalf("Ready() expected error")
	}

	if !errors.Is(err, downErr) {
		t.Fatalf("Ready() error = %v, want wrapped %v", err, downErr)
	}
}

func TestServiceReadyDraining(t *testing.T) {
	ctrl := gomock.NewController(t)
	probe := NewMockProbe(ctrl)

	svc := New(probe)
	svc.StartDrain()

	err := svc.Ready(context.Background())
	if !errors.Is(err, ErrDraining) {
		t.Fatalf("Ready() error = %v, want ErrDraining", err)
	}
}
