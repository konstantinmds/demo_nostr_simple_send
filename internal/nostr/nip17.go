package nostr

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip44"
)

var (
	ErrNIP17IncompatibleRelay = errors.New("relay does not support NIP-17")
)

// generateRandomPrivateKey generates a random private key for ephemeral usage
func generateRandomPrivateKey() (string, error) {
	var key [32]byte
	_, err := rand.Read(key[:])
	if err != nil {
		return "", err
	}
	// Convert to hex string
	return hex.EncodeToString(key[:]), nil
}

// randomTimePast returns a random timestamp up to 2 days in the past
func randomTimePast() nostr.Timestamp {
	now := time.Now().Unix()
	// Random time up to 2 days in the past (172800 seconds)
	max := big.NewInt(172800)
	n, _ := rand.Int(rand.Reader, max)
	randomOffset := time.Duration(n.Int64()) * time.Second
	randomTime := time.Unix(now, 0).Add(-randomOffset)
	return nostr.Timestamp(randomTime.Unix())
}

// NIP17DirectMessage creates an encrypted NIP-17 direct message (kind 14)
func NIP17DirectMessage(content string, recipientPubKeys []string, replyToID string, subject string) *nostr.Event {
	// Create the unsigned kind 14 event
	ev := &nostr.Event{
		Kind:      14, // Chat message
		Content:   content,
		CreatedAt: nostr.Now(),
		Tags:      nostr.Tags{},
	}

	// Add recipient p tags
	for _, pubKey := range recipientPubKeys {
		ev.Tags = append(ev.Tags, nostr.Tag{"p", pubKey})
	}

	// Add reply tag if this is a reply
	if replyToID != "" {
		ev.Tags = append(ev.Tags, nostr.Tag{"e", replyToID})
	}

	// Add subject if provided
	if subject != "" {
		ev.Tags = append(ev.Tags, nostr.Tag{"subject", subject})
	}

	return ev
}

// sealEvent takes an unsigned event (kind 14 or 15) and seals it (kind 13)
func sealEvent(unsignedEvent *nostr.Event, senderPrivateKey, receiverPubKey string) (*nostr.Event, error) {
	// Get sender's public key
	senderPubKey, err := GetPublicKeyFromPrivate(senderPrivateKey)
	if err != nil {
		return nil, err
	}

	// Set the pubkey in the unsigned event to the sender's pubkey
	unsignedEvent.PubKey = senderPubKey

	// Generate conversation key from sender's private key and receiver's public key
	conversationKey, err := nip44.GenerateConversationKey(receiverPubKey, senderPrivateKey)
	if err != nil {
		return nil, err
	}

	// Serialize unsigned event to JSON
	eventJSON, err := json.Marshal(unsignedEvent)
	if err != nil {
		return nil, err
	}

	// Encrypt the unsigned event content with NIP-44
	encryptedContent, err := nip44.Encrypt(string(eventJSON), conversationKey)
	if err != nil {
		return nil, err
	}

	// Create the sealed event (kind 13)
	sealedEvent := &nostr.Event{
		Kind:      13, // Sealed event
		Content:   encryptedContent,
		CreatedAt: randomTimePast(), // Random time to avoid correlation
		PubKey:    senderPubKey,
		Tags:      nostr.Tags{},
	}

	// Sign the sealed event
	err = sealedEvent.Sign(senderPrivateKey)
	if err != nil {
		return nil, err
	}

	return sealedEvent, nil
}

// giftWrapEvent takes a sealed event and gift wraps it (kind 1059)
func giftWrapEvent(sealedEvent *nostr.Event, receiverPubKey string) (*nostr.Event, error) {
	// Generate random private key for the gift wrap
	randomPrivateKey, err := generateRandomPrivateKey()
	if err != nil {
		return nil, err
	}

	// Get the public key for the random private key
	randomPubKey, err := GetPublicKeyFromPrivate(randomPrivateKey)
	if err != nil {
		return nil, err
	}

	// Generate conversation key from random private key and receiver's public key
	conversationKey, err := nip44.GenerateConversationKey(receiverPubKey, randomPrivateKey)
	if err != nil {
		return nil, err
	}

	// Serialize sealed event to JSON
	eventJSON, err := json.Marshal(sealedEvent)
	if err != nil {
		return nil, err
	}

	// Encrypt the sealed event with NIP-44
	encryptedSealedEvent, err := nip44.Encrypt(string(eventJSON), conversationKey)
	if err != nil {
		return nil, err
	}

	// Create the gift wrap event (kind 1059)
	giftWrapEvent := &nostr.Event{
		Kind:      1059, // Gift wrap
		Content:   encryptedSealedEvent,
		CreatedAt: randomTimePast(), // Random time to avoid correlation
		PubKey:    randomPubKey,
		Tags:      nostr.Tags{nostr.Tag{"p", receiverPubKey}},
	}

	// Sign the gift wrap with the random private key
	err = giftWrapEvent.Sign(randomPrivateKey)
	if err != nil {
		return nil, err
	}

	return giftWrapEvent, nil
}

// getPreferredNIP17Relays fetches recipient's preferred DM relays
func getPreferredNIP17Relays(ctx context.Context, pubKey string, knownRelays []string) ([]string, error) {
	// Try to find the user's kind 10050 events
	preferredRelays := []string{}

	// First check known relays
	for _, relayURL := range knownRelays {
		relay, err := nostr.RelayConnect(ctx, relayURL)
		if err != nil {
			continue // Skip this relay, try the next one
		}
		defer relay.Close()

		sub, err := relay.Subscribe(ctx, nostr.Filters{
			{
				Kinds:   []int{10050},
				Authors: []string{pubKey},
				Limit:   1,
			},
		})
		if err != nil {
			continue // Skip if subscription fails
		}

		for ev := range sub.Events {
			for _, tag := range ev.Tags {
				if len(tag) >= 2 && tag[0] == "relay" {
					preferredRelays = append(preferredRelays, tag[1])
				}
			}
			// If we found relays, we're done
			if len(preferredRelays) > 0 {
				return preferredRelays, nil
			}
		}
	}

	// If we didn't find any preferred relays, the user might not be ready for NIP-17
	if len(preferredRelays) == 0 {
		return nil, ErrNIP17IncompatibleRelay
	}

	return preferredRelays, nil
}

// SendNIP17DirectMessage sends a private direct message using NIP-17
func SendNIP17DirectMessage(ctx context.Context, privateKey string, recipientKeys []string,
	message string, relayURLs []string, replyToID, subject, clientID string) (string, error) {

	// Get sender's public key
	senderPubKey, err := GetPublicKeyFromPrivate(privateKey)
	if err != nil {
		return "", err
	}

	// Create unsigned kind 14 event
	unsignedDM := NIP17DirectMessage(message, recipientKeys, replyToID, subject)

	// Add client tag if provided
	if clientID != "" {
		unsignedDM.Tags = append(unsignedDM.Tags, nostr.Tag{"client", clientID})
	}

	// Track the gift-wrapped events we create
	giftWraps := []*nostr.Event{}

	// Create gift wraps for each recipient
	for _, recipientKey := range recipientKeys {
		// Decode recipient's key if in NIP-19 format
		recipientPubKey, err := DecodePublicKey(recipientKey)
		if err != nil {
			return "", err
		}

		// Create sealed event
		sealedEvent, err := sealEvent(unsignedDM, privateKey, recipientPubKey)
		if err != nil {
			return "", err
		}

		// Create gift wrap
		giftWrap, err := giftWrapEvent(sealedEvent, recipientPubKey)
		if err != nil {
			return "", err
		}

		giftWraps = append(giftWraps, giftWrap)
	}

	// Also create a gift wrap for the sender (so they can see their own messages)
	sealedForSender, err := sealEvent(unsignedDM, privateKey, senderPubKey)
	if err != nil {
		return "", err
	}

	senderGiftWrap, err := giftWrapEvent(sealedForSender, senderPubKey)
	if err != nil {
		return "", err
	}
	giftWraps = append(giftWraps, senderGiftWrap)

	// For each recipient, try to find their preferred relays
	recipientRelays := make(map[string][]string)

	for _, recipientKey := range recipientKeys {
		recipientPubKey, _ := DecodePublicKey(recipientKey)
		// Try to get preferred relays
		preferredRelays, err := getPreferredNIP17Relays(ctx, recipientPubKey, relayURLs)
		if err != nil {
			// If no preferred relays, fall back to provided relays
			recipientRelays[recipientPubKey] = relayURLs
		} else {
			recipientRelays[recipientPubKey] = preferredRelays
		}
	}

	// Also add sender's relays (for their own copy)
	recipientRelays[senderPubKey] = relayURLs

	// Publish each gift wrap to the appropriate relays
	var publishErr error
	var firstNoteID string

	for i, giftWrap := range giftWraps {
		// Get the recipient from the p tag
		var recipient string
		for _, tag := range giftWrap.Tags {
			if len(tag) >= 2 && tag[0] == "p" {
				recipient = tag[1]
				break
			}
		}

		// Use appropriate relays for this recipient
		recipientRelayList := recipientRelays[recipient]
		if len(recipientRelayList) == 0 {
			continue // Skip if no relays for this recipient
		}

		// Publish to each relay for this recipient
		for _, relayURL := range recipientRelayList {
			relay, err := nostr.RelayConnect(ctx, relayURL)
			if err != nil {
				publishErr = err
				continue // Try next relay
			}

			err = relay.Publish(ctx, *giftWrap)
			relay.Close()

			if err != nil {
				publishErr = err
				continue // Try next relay
			}

			// Store the first successfully published note ID
			if i == 0 && firstNoteID == "" {
				noteID, err := nip19.EncodeNote(giftWrap.ID)
				if err == nil {
					firstNoteID = noteID
				}
			}
		}
	}

	// If we never published successfully, return the error
	if firstNoteID == "" && publishErr != nil {
		return "", publishErr
	}

	return firstNoteID, nil
}

// PublishNIP17Preferences publishes the user's NIP-17 preferred relays
func PublishNIP17Preferences(ctx context.Context, privateKey string, preferredRelayURLs []string) (string, error) {
	// Get sender's public key
	pubKey, err := GetPublicKeyFromPrivate(privateKey)
	if err != nil {
		return "", err
	}

	// Create the event
	ev := nostr.Event{
		PubKey:    pubKey,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      10050, // NIP-17 preferred relays
		Tags:      nostr.Tags{},
		Content:   "",
	}

	// Add relay tags
	for _, relayURL := range preferredRelayURLs {
		ev.Tags = append(ev.Tags, nostr.Tag{"relay", relayURL})
	}

	// Sign the event
	err = ev.Sign(privateKey)
	if err != nil {
		return "", err
	}

	// Publish to all provided relays
	var publishErr error
	published := false

	for _, relayURL := range preferredRelayURLs {
		relay, err := nostr.RelayConnect(ctx, relayURL)
		if err != nil {
			publishErr = err
			continue // Try next relay
		}
		defer relay.Close()

		err = relay.Publish(ctx, ev)
		if err != nil {
			publishErr = err
			continue // Try next relay
		}

		published = true
	}

	if !published && publishErr != nil {
		return "", publishErr
	}

	// Get note ID in bech32 format
	noteID, err := nip19.EncodeNote(ev.ID)
	if err != nil {
		return "", err
	}

	return noteID, nil
}
