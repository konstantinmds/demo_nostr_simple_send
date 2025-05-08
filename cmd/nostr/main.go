package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
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
	dmPrivateKeyHex := cmdDM.String("key", "", "Private key in hex format")
	dmNsecKey := cmdDM.String("nsec", "", "Private key in nsec format")
	dmRecipient := cmdDM.String("to", "", "Recipient public key (hex, npub, or nprofile)")
	dmMessage := cmdDM.String("message", "", "Message content to send")
	dmRelayURL := cmdDM.String("relay", "wss://relay.damus.io", "Relay URL")
	dmTimeout := cmdDM.Duration("timeout", 5*time.Second, "Timeout for relay operations")
	dmClientID := cmdDM.String("client", "nostr_demo_golang", "Client identifier")

	// NIP-17 Direct Message Command
	nip17dmCmd := flag.NewFlagSet("nip17dm", flag.ExitOnError)
	nip17dmPrivKeyHex := nip17dmCmd.String("key", "", "Private key in hex format")
	nip17dmNsecKey := nip17dmCmd.String("nsec", "", "Private key in nsec format")
	nip17dmRecipients := nip17dmCmd.String("to", "", "Comma-separated list of recipient public keys")
	nip17dmMessage := nip17dmCmd.String("message", "", "Message content to send")
	nip17dmRelayURLs := nip17dmCmd.String("relays", "wss://relay.damus.io", "Comma-separated list of relay URLs")
	nip17dmReplyTo := nip17dmCmd.String("reply-to", "", "Event ID to reply to")
	nip17dmSubject := nip17dmCmd.String("subject", "", "Subject/title of conversation")
	nip17dmTimeout := nip17dmCmd.Duration("timeout", 5*time.Second, "Timeout for relay operations")
	nip17dmClientID := nip17dmCmd.String("client", "nostr_demo_golang", "Client identifier")

	// NIP-17 Set Preferred Relays Command
	nip17relaysCmd := flag.NewFlagSet("nip17relays", flag.ExitOnError)
	nip17relaysPrivKeyHex := nip17relaysCmd.String("key", "", "Private key in hex format")
	nip17relaysNsecKey := nip17relaysCmd.String("nsec", "", "Private key in nsec format")
	nip17relaysURLs := nip17relaysCmd.String("relays", "", "Comma-separated list of preferred relay URLs")
	nip17relaysTimeout := nip17relaysCmd.Duration("timeout", 5*time.Second, "Timeout for relay operations")

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

	case "nip17dm":
		nip17dmCmd.Parse(os.Args[2:])
		handleNIP17DMCommand(nip17dmPrivKeyHex, nip17dmNsecKey, nip17dmRecipients, nip17dmMessage, nip17dmRelayURLs, nip17dmReplyTo, nip17dmSubject, nip17dmClientID, nip17dmTimeout)

	case "nip17relays":
		nip17relaysCmd.Parse(os.Args[2:])
		handleNIP17RelaysCommand(nip17relaysPrivKeyHex, nip17relaysNsecKey, nip17relaysURLs, nip17relaysTimeout)

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

func handleNIP17DMCommand(privateKeyHex, nsecKey, recipients, message, relayURLs, replyToID, subject, clientID *string, timeout *time.Duration) {
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

	// Parse recipient list
	recipientList := strings.Split(*recipients, ",")
	if len(recipientList) == 0 || (len(recipientList) == 1 && recipientList[0] == "") {
		fmt.Println("Error: No recipients specified")
		os.Exit(1)
	}

	// Parse relay URLs
	relayList := strings.Split(*relayURLs, ",")
	if len(relayList) == 0 || (len(relayList) == 1 && relayList[0] == "") {
		fmt.Println("Error: No relay URLs specified")
		os.Exit(1)
	}

	// Display info about the recipients
	for i, recipient := range recipientList {
		recipientHex, err := nostr.DecodePublicKey(recipient)
		if err != nil {
			fmt.Printf("Error with recipient key #%d: %v\n", i+1, err)
			os.Exit(1)
		}

		recipientNpub, _ := nostr.FormatPublicKey(recipientHex)
		fmt.Printf("Sending encrypted message to: %s\n", recipientNpub)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Send the NIP-17 DM
	fmt.Printf("Sending NIP-17 encrypted DM via %d relays...\n", len(relayList))
	noteID, err := nostr.SendNIP17DirectMessage(ctx, privateKey, recipientList, *message, relayList, *replyToID, *subject, *clientID)
	if err != nil {
		fmt.Printf("Error sending NIP-17 DM: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("NIP-17 direct message sent successfully!")
	fmt.Printf("Event ID: %s\n", noteID)
	fmt.Printf("View at: https://njump.me/%s\n", noteID)
}

func handleNIP17RelaysCommand(privateKeyHex, nsecKey, relayURLs *string, timeout *time.Duration) {
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

	// Parse relay URLs
	relayList := strings.Split(*relayURLs, ",")
	if len(relayList) == 0 || (len(relayList) == 1 && relayList[0] == "") {
		fmt.Println("Error: No relay URLs specified")
		os.Exit(1)
	}

	fmt.Printf("Setting NIP-17 preferred relays: %s\n", *relayURLs)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Publish NIP-17 preferences
	noteID, err := nostr.PublishNIP17Preferences(ctx, privateKey, relayList)
	if err != nil {
		fmt.Printf("Error publishing NIP-17 preferences: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("NIP-17 relay preferences published successfully!")
	fmt.Printf("Event ID: %s\n", noteID)
	fmt.Printf("View at: https://njump.me/%s\n", noteID)
}
