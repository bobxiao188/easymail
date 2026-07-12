package milter

import (
	"errors"
)

// pre-defined errors
var (
	errCloseSession = errors.New("stop current milter processing")
)
