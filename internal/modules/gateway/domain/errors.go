package domain

import "errors"

var ErrSaturated = errors.New("gateway concurrency limit reached")
