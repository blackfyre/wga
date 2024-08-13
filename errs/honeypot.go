package errs

import "errors"

var ErrHoneypotTriggered = errors.New("honeypot triggered")
