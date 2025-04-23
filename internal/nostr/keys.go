package nostr

import (
	"errors"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

// DeterminePrivateKey returns the hex private key from either nsec or hex format
func DeterminePrivateKey(privateKeyHex, nsecKey string) (string, error) {
	if nsecKey != "" {
		// Convert nsec to hex
		prefix, decoded, err := nip19.Decode(nsecKey)
		if err != nil {
			return "", err
		}
		if prefix != "nsec" {
			return "", errors.New("invalid nsec key format")
		}
		return decoded.(string), nil
	} else if privateKeyHex != "" {
		return privateKeyHex, nil
	}
	return "", errors.New("no private key provided")
}

func GetPublicKeyFromPrivate(privateKeyHex string) (string, error) {
	return nostr.GetPublicKey(privateKeyHex)
}

func DecodePublicKey(pubKey string) (string, error) {
	// If it's already a hex key, return it
	if len(pubKey) == 64 {
		return pubKey, nil
	}

	// Try to decode as NIP-19 format
	if len(pubKey) > 0 {
		prefix, decoded, err := nip19.Decode(pubKey)
		if err != nil {
			return "", err
		}

		switch prefix {
		case "npub":
			return decoded.(string), nil

		case "nprofile":
			switch v := decoded.(type) {
			case nostr.ProfilePointer:
				return v.PublicKey, nil
			case *nostr.ProfilePointer:
				return v.PublicKey, nil
			default:
				return "", errors.New("unsupported nprofile format")
			}
		}
	}

	return "", errors.New("unsupported public key format")
}

// FormatPublicKey converts a hex public key to bech32 npub format
func FormatPublicKey(pubKeyHex string) (string, error) {
	return nip19.EncodePublicKey(pubKeyHex)
}
