// Command genauthzkey generates an Ed25519 keypair for SWM device authorization.
//
// Run this ONCE per environment (or per key rotation), not on every build:
//   - The PRIVATE key goes into the server env / secrets manager (AUTHZ_*).
//   - The matching PUBLIC key must be embedded into the client build
//     (SwmConfig.AuthzPublicKeys) under the same key_id.
//
// Regenerating the key invalidates every already-distributed client that embeds
// the old public key, so treat the output as a long-lived secret.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	keyID := flag.String("key-id", "", "key id (default: authz-YYYYMMDD)")
	flag.Parse()

	id := *keyID
	if id == "" {
		id = "authz-" + time.Now().UTC().Format("20060102")
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, "keygen failed:", err)
		os.Exit(1)
	}

	fmt.Printf("# SWM device-authorization keypair  (key_id=%s)\n", id)
	fmt.Println("#")
	fmt.Println("# 1) SERVER secret — put in env / secrets manager, NEVER commit:")
	fmt.Printf("AUTHZ_KEY_ID=%s\n", id)
	fmt.Printf("AUTHZ_SIGNING_PRIVATE_KEY=%s\n", hex.EncodeToString(priv.Seed()))
	fmt.Println("#")
	fmt.Println("# 2) CLIENT public key — add to SwmConfig.AuthzPublicKeys (Services/SwmConfig.cs):")
	fmt.Printf("#    [\"%s\"] = \"%s\",\n", id, hex.EncodeToString(pub))
}
