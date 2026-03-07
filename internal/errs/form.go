package errs

import "errors"

var ErrMessageRequired = errors.New("message required")
var ErrRecaptchaTokenRequired = errors.New("recaptcha token required")
