package nostr

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidKeyFormat    = errors.New("invalid key format")
	ErrNoRelayConnected    = errors.New("no relay connected, call ConnectToRelay first")
)

type RelayError struct {
	RelayURL string
	Err      error
}

func (e *RelayError) Error() string {
	return fmt.Sprintf("relay %s: %v", e.RelayURL, e.Err)
}
