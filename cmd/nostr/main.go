package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/konstantinmds/nostr_demo_golang/internal/nostr"
)

func main() {
	_ = godotenv.Load()

	var (
		cmdPost = flag.NewFlagSet("post", flag.ExitOnError)
		cmdDM   = flag.NewFlagSet("dm", flag.ExitOnError)
	)

	// Post command flags
	postPrivateKeyHex := cmdPost.String("key", os.Getenv("NOSTR_PRIVATE_KEY"), "Private key in hex format")
	postNsecKey := cmdPost.String("nsec", os.Getenv("NOSTR_NSEC_KEY"), "Private key in nsec format")
	postMessage := cmdPost.String("message", "Hello world!", "Message to post")
	postRelayURL := cmdPost.String("relay", "wss://relay.damus.io", "Relay URL")
	postClientID := cmdPost.String("client", "nostr_demo_golang", "Client identifier")
	postTags := cmdPost.String("tags", "", "Additional tags in format 'key1:value1,key2:value2'")
	postTimeout := cmdPost.Duration("timeout", 5*time.Second, "Connection timeout")

	// DM command flags
	dmPrivateKeyHex := cmdDM.String("key", os.Getenv("NOSTR_PRIVATE_KEY"), "Private key in hex format")
	dmNsecKey := cmdDM.String("nsec", os.Getenv("NOSTR_NSEC_KEY"), "Private key in nsec format")
	dmRecipient := cmdDM.String("to", "", "Recipient's public key (hex, npub, or nprofile)")
	dmMessage := cmdDM.String("message", "", "Message to send")
	dmRelayURL := cmdDM.String("relay", "wss://relay.damus.io", "Relay URL")
	dmClientID := cmdDM.String("client", "nostr_demo_golang", "Client identifier")
	dmTimeout := cmdDM.Duration("timeout", 5*time.Second, "Connection timeout")

	switch os.Args[1] {
	case "post":
		cmdPost.Parse(os.Args[2:])
		handlePostCommand(postPrivateKeyHex, postNsecKey, postMessage, postRelayURL, postClientID, postTags, postTimeout)

	case "dm":
		cmdDM.Parse(os.Args[2:])
		if *dmRecipient == "" {
			fmt.Println("Error: recipient is required for direct messages")
			os.Exit(1)
		}
		if *dmMessage == "" {
			fmt.Println("Error: message is required for direct messages")
			os.Exit(1)
		}
		handleDMCommand(dmPrivateKeyHex, dmNsecKey, dmRecipient, dmMessage, dmRelayURL, dmClientID, dmTimeout)

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Run 'nostr -h' for usage information")
		os.Exit(1)
	}
}

func handlePostCommand(privateKeyHex, nsecKey, message, relayURL, clientID, tags *string, timeout *time.Duration) {
	privateKey, err := nostr.DeterminePrivateKey(*privateKeyHex, *nsecKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	pubKey, err := nostr.GetPublicKeyFromPrivate(privateKey)
	if err != nil {
		fmt.Printf("Error getting public key: %v\n", err)
		os.Exit(1)
	}

	// Convert to bech32 for display
	npub, err := nostr.FormatPublicKey(pubKey)
	if err != nil {
		fmt.Printf("Warning: couldn't format public key: %v\n", err)
	} else {
		fmt.Printf("Using public key: %s\n", npub)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	fmt.Printf("Sending post to %s...\n", *relayURL)
	noteID, err := nostr.SendPublicPost(ctx, privateKey, *message, *relayURL, *clientID, *tags)
	if err != nil {
		fmt.Printf("Error sending post: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Post sent successfully!")
	fmt.Printf("Event ID: %s\n", noteID)
	fmt.Printf("View at: https://njump.me/%s\n", noteID)
}

func handleDMCommand(privateKeyHex, nsecKey, recipient, message, relayURL, clientID *string, timeout *time.Duration) {
	privateKey, err := nostr.DeterminePrivateKey(*privateKeyHex, *nsecKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	pubKey, err := nostr.GetPublicKeyFromPrivate(privateKey)
	if err != nil {
		fmt.Printf("Error getting public key: %v\n", err)
		os.Exit(1)
	}

	npub, err := nostr.FormatPublicKey(pubKey)
	if err != nil {
		fmt.Printf("Warning: couldn't format public key: %v\n", err)
	} else {
		fmt.Printf("Using public key: %s\n", npub)
	}

	recipientHex, err := nostr.DecodePublicKey(*recipient)
	if err != nil {
		fmt.Printf("Error with recipient key: %v\n", err)
		os.Exit(1)
	}

	recipientNpub, _ := nostr.FormatPublicKey(recipientHex)
	fmt.Printf("Sending encrypted message to: %s\n", recipientNpub)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Send the DM
	fmt.Printf("Sending encrypted DM via %s...\n", *relayURL)
	noteID, err := nostr.SendDirectMessage(ctx, privateKey, recipientHex, *message, *relayURL, *clientID)
	if err != nil {
		fmt.Printf("Error sending DM: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Direct message sent successfully!")
	fmt.Printf("Event ID: %s\n", noteID)
	fmt.Printf("View at: https://njump.me/%s\n", noteID)
}
