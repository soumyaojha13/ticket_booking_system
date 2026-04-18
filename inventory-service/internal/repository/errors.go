package repository

import "errors"

var (
	ErrSeatNotFound          = errors.New("seat not found")
	ErrSeatAlreadyLocked     = errors.New("seat already locked")
	ErrSeatLockedByOtherUser = errors.New("seat is locked by another user")
	ErrSeatNotAvailable      = errors.New("seat is not available")
)
