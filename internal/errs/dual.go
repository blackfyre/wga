package errs

import "errors"

var ErrUnknownDualPane = errors.New("unknown dual pane")
var ErrTooManyParts = errors.New("too many parts")
var ErrUnsupportedPaneType = errors.New("unsupported pane type")
