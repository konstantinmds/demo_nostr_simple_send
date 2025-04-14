package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func main() {
	_ = godotenv.Load()

	var (
		privateKeyHex = flag.String("key", os.Getenv("NOSTR_PRIVATE_KEY"), "Private key in hex format")
		nsecKey       = flag.String("nsec", os.Getenv("NOSTR_NSEC_KEY"), "Private key in nsec format")
		message       = flag.String("message", "Hello world!", "Message to post")
		relayURL      = flag.String("relay", "wss://relay.damus.io", "Relay URL")
		clientID      = flag.String("client", "nostr_demo_golang", "Client identifier")
		tags          = flag.String("tags", "", "Additional tags")
		timeout       = flag.Duration("timeout", 5*time.Second, "Connection timeout")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Determine private key
	var sk string
	if *nsecKey != "" {
		// Convert nsec to hex
		_, decoded, err := nip19.Decode(*nsecKey)
		if err != nil {
			log.Fatalf("Error decoding nsec key: %v", err)
		}
		sk = decoded.(string)
	} else if *privateKeyHex != "" {
		sk = *privateKeyHex
	} else {
		log.Fatalf("No private key provided.")
	}

	// Get public key from private key
	pubKey, err := nostr.GetPublicKey(sk)
	if err != nil {
		log.Fatalf("Error getting public key: %v", err)
	}

	// Convert to bech32 format
	npub, err := nip19.EncodePublicKey(pubKey)
	if err != nil {
		log.Fatalf("Error encoding public key: %v", err)
	}

	fmt.Printf("Using public key: %s\n", npub)

	// Create event
	ev := nostr.Event{
		PubKey:    pubKey,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{},
		Content:   *message,
	}

	// Add client tag
	ev.Tags = append(ev.Tags, nostr.Tag{"client", *clientID})

	// Process any custom tags
	if *tags != "" {
		// Split by commas to get key:value pairs
		pairs := strings.Split(*tags, ",")
		for _, pair := range pairs {
			// Split each pair by colon
			kv := strings.Split(pair, ":")
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				
				// Add decoding logic for NIP-19 formats
				if strings.HasPrefix(value, "npub") || strings.HasPrefix(value, "nprofile") {
					_, decoded, err := nip19.Decode(value)
					if err != nil {
						log.Fatalf("Error decoding %s: %v", value, err)
					}
					// Handle different decoded types
					switch v := decoded.(type) {
					case string:
						value = v
					case *nostr.ProfilePointer:
						// For nprofile format
						value = v.PublicKey
					case nostr.ProfilePointer:
						// Alternate form of ProfilePointer
						value = v.PublicKey
					default:
						log.Fatalf("Unsupported NIP-19 type: %T for value %s", decoded, value)
					}
					
					fmt.Printf("Decoded %s to hex: %s\n", kv[1], value)
				}
				
				ev.Tags = append(ev.Tags, nostr.Tag{key, value})
			}
		}
	}

	// Sign event
	err = ev.Sign(sk)
	if err != nil {
		log.Fatalf("Error signing event: %v", err)
	}

	// Verify the signature
	ok, err := ev.CheckSignature()
	if err != nil {
		log.Fatalf("Error checking signature: %v", err)
	}
	if !ok {
		log.Fatalf("Invalid signature")
	}
	fmt.Println("Signature verified successfully")

	// Connect to relay
	fmt.Printf("Connecting to relay: %s\n", *relayURL)
	relay, err := nostr.RelayConnect(ctx, *relayURL)
	if err != nil {
		log.Fatalf("Error connecting to relay: %v", err)
	}
	defer relay.Close()

	// Publish event
	fmt.Println("Publishing event...")
	err = relay.Publish(ctx, ev)
	if err != nil {
		log.Fatalf("Error publishing event: %v", err)
	}
	
	fmt.Printf("Event published successfully to: %s\n", relay.URL)

	// Get note ID in bech32 format
	noteID, err := nip19.EncodeNote(ev.ID)
	if err != nil {
		log.Printf("Warning: couldn't encode note ID: %v", err)
	}

	fmt.Println("Event published successfully!")
	fmt.Printf("Event ID: %s\n", noteID)
	fmt.Printf("View at: https://njump.me/%s\n", noteID)
} 