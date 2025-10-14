package app

import "github.com/michiwend/gomusicbrainz"

type Config struct {
	// User input fields
	InputDir     string
	OutputDir    string
	DiscogsToken string
	Copy         bool
	// Private dependencies
	mbClient *gomusicbrainz.WS2Client
}
