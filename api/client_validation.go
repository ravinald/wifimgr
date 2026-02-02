package api

import (
	"fmt"
)

// This file contains additional token validation helpers
// The main ValidateAPIToken and GetAPIUserInfo methods are in client_config.go

// ErrUnauthorized is a sentinel error for unauthorized requests
var ErrUnauthorized = fmt.Errorf("unauthorized: invalid API token")

// ErrNotFound is a sentinel error for resource not found
var ErrNotFound = fmt.Errorf("not found: resource does not exist")
