package metertest

import "context"

type Meter struct {
}

func NewMeter() *Meter {
	return &Meter{}
}

func (m *Meter) Count(ctx context.Context, name string, desc string, t string) error {
	return nil
}
