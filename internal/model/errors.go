package model

import "errors"

// Sentinel errors shared across adapters and the probe engine.
var (
	// ErrNoPrivateKey is returned by MintToken when the issuer does not control
	// its signing key (SaaS IdPs). Probes 1, 8, 10, 13 and others requiring a
	// mint are unavailable in that case (PRD §1.3).
	ErrNoPrivateKey = errors.New("issuer does not control its private key")

	// ErrUnsupported is returned by adapter operations that an issuer type cannot
	// perform (e.g. AnnounceKey on a generic issuer without admin access).
	ErrUnsupported = errors.New("operation unsupported for this issuer")

	// ErrNoAnnouncedKey is returned by the canary probe (13) when no key is in the
	// announced state.
	ErrNoAnnouncedKey = errors.New("no key in announced state")

	// ErrNotFound is a generic lookup miss from the store.
	ErrNotFound = errors.New("not found")
)
