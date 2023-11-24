package models

import "context"

type WaitKey struct {
	Value   string
	Timeout context.Context
}
