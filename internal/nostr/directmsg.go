package nostr

import (
	"context"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"github.com/nbd-wtf/go-nostr/nip19"
)

// EncryptDirectMessage encrypts a message using NIP-04
func EncryptDirectMessage(message, recipientPubKey, privateKey string) (string, error) {
	// Compute shared secret
	sharedSecret, err := nip04.ComputeSharedSecret(recipientPubKey, privateKey)
	if err != nil {
		return "", err
	}

	// Encrypt the message
	encrypted, err := nip04.Encrypt(message, sharedSecret)
	if err != nil {
		return "", err
	}

	return encrypted, nil
}

// SendDirectMessage encrypts and sends a direct message to a recipient
func SendDirectMessage(ctx context.Context, privateKey, recipientKey, message, relayURL, clientID string) (string, error) {
	// Get sender's public key
	pubKey, err := GetPublicKeyFromPrivate(privateKey)
	if err != nil {
		return "", err
	}

	// Decode recipient's key if in NIP-19 format
	recipientPubKey, err := DecodePublicKey(recipientKey)
	if err != nil {
		return "", err
	}

	// Encrypt the message using our NIP-04 implementation
	encryptedContent, err := EncryptDirectMessage(message, recipientPubKey, privateKey)
	if err != nil {
		return "", ErrEncryptionFailed
	}

	// Create the event
	ev := nostr.Event{
		PubKey:    pubKey,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      nostr.KindEncryptedDirectMessage, // Kind 4 for encrypted DMs
		Tags:      nostr.Tags{},
		Content:   encryptedContent,
	}

	// Add the recipient as a 'p' tag (required for DMs)
	ev.Tags = append(ev.Tags, nostr.Tag{"p", recipientPubKey})

	// Add client tag
	ev.Tags = append(ev.Tags, nostr.Tag{"client", clientID})

	// not sure are the tags needed for me here :/ but leave it here for now

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
