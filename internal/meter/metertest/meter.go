package metertest

import (
	"context"
	"time"
)

type Meter struct{}

func NewMeter() *Meter {
	return &Meter{}
}

func (m *Meter) Count(ctx context.Context, name string, desc string, t string) error {
	return nil
}

func (m *Meter) Histogram(ctx context.Context, op string, start time.Time) error {
	return nil
}
