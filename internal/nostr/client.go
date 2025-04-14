package nostr

import (
	"context"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

// Client represents a simplified Nostr client 
type Client struct {
	sk      string
	pubKey  string
	npub    string
	relay   *nostr.Relay
	timeout time.Duration
}

func NewClient(privateKeyHex string, timeout time.Duration) (*Client, error) {
	pubKey, err := nostr.GetPublicKey(privateKeyHex)
	if err != nil {
		return nil, err
	}

	// Convert to npub format
	npub, err := nip19.EncodePublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		sk:      privateKeyHex,
		pubKey:  pubKey,
		npub:    npub,
		relay:   nil,
		timeout: timeout,
	}, nil
}

// NewClientFromNsec creates a new Nostr client from an nsec private key
func NewClientFromNsec(nsecKey string, timeout time.Duration) (*Client, error) {
	prefix, decoded, err := nip19.Decode(nsecKey)
	if err != nil {
		return nil, err
	}
	if prefix != "nsec" {
		return nil, ErrInvalidKeyFormat
	}
	return NewClient(decoded.(string), timeout)
}

// GetPublicKey returns the client's public key in hex format
func (c *Client) GetPublicKey() string {
	return c.pubKey
}

// GetPublicKeyBech32 returns the client's public key in bech32 (npub) format
func (c *Client) GetPublicKeyBech32() string {
	return c.npub
}

func (c *Client) ConnectToRelay(ctx context.Context, url string) error {
	// Close existing connection if any
	if c.relay != nil {
		c.relay.Close()
	}

	relay, err := nostr.RelayConnect(ctx, url)
	if err != nil {
		return err
	}

	c.relay = relay
	return nil
}

func (c *Client) CreateTextNote(content string, tags [][]string) (*nostr.Event, error) {
	ev := nostr.Event{
		PubKey:    c.pubKey,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{},
		Content:   content,
	}

	if len(tags) > 0 {
		for _, tag := range tags {
			ev.Tags = append(ev.Tags, tag)
		}
	}

	err := ev.Sign(c.sk)
	if err != nil {
		return nil, err
	}

	return &ev, nil
}

func (c *Client) PublishEvent(ctx context.Context, event *nostr.Event) error {
	if c.relay == nil {
		return ErrNoRelayConnected
	}
	
	return c.relay.Publish(ctx, *event)
}

func (c *Client) Close() {
	if c.relay != nil {
		c.relay.Close()
		c.relay = nil
	}
} 