package nostr

import (
	"context"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

// SendPublicPost sends a public post to a relay
func SendPublicPost(ctx context.Context, privateKey, message, relayURL, clientID, tags string) (string, error) {
	// Get public key from private key
	pubKey, err := GetPublicKeyFromPrivate(privateKey)
	if err != nil {
		return "", err
	}

	// Create the event
	ev := nostr.Event{
		PubKey:    pubKey,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{},
		Content:   message,
	}

	// Add client tag
	ev.Tags = append(ev.Tags, nostr.Tag{"client", clientID})

	// Process custom tags
	if tags != "" {
		// Split by commas to get key:value pairs
		pairs := strings.Split(tags, ",")
		for _, pair := range pairs {
			// Split each pair by colon
			kv := strings.Split(pair, ":")
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				
				// Handle NIP-19 format keys
				if key == "p" && (strings.HasPrefix(value, "npub") || strings.HasPrefix(value, "nprofile")) {
					decodedKey, err := DecodePublicKey(value)
					if err != nil {
						return "", err
					}
					value = decodedKey
				}
				
				ev.Tags = append(ev.Tags, nostr.Tag{key, value})
			}
		}
	}

	// Sign the event
	err = ev.Sign(privateKey)
	if err != nil {
		return "", err
	}

	// Connect to relay
	relay, err := nostr.RelayConnect(ctx, relayURL)
	if err != nil {
		return "", err
	}
	defer relay.Close()

	// Publish the event
	err = relay.Publish(ctx, ev)
	if err != nil {
		return "", err
	}

	// Get note ID in bech32 format
	noteID, err := nip19.EncodeNote(ev.ID)
	if err != nil {
		return "", err
	}

	return noteID, nil
} 