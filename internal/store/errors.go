package store

import "errors"

var ErrNotFound = errors.New("store: not found")
var ErrDuplicateName = errors.New("store: duplicate name")
