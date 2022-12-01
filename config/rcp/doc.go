// Package rcp contains Relay Config Provider Clients used to fetch the Proposer Config from different RCPs.
//
// Relay Config Providers are data sources which hold information about proposers relays and the default relays.
// For example an RCP maybe a file containing Proposer Config, or a remote API endpoint returning the Proposer Config.
//
// A Proposer Config is basically a JSON structure which has the information about the proposers and their relays.
// The schema is a subject to change as it is still being discussed at https://github.com/flashbots/mev-boost/issues/154
// An example of a valid Proposer Config maybe found at /testdata/valid-proposer-config.json.
package rcp
