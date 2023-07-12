// Package rcm implements Relay Configuration Manager (RCM) which is used to manage relays and their configuration.
//
// RCM (Configurator) maintains a RelayRegistry of all the registered validators and their corresponding relay configs.
//
// When a new instance of Configurator is created via New, SyncConfig is called to synchronise RelayRegistry contents
// with an RCP via RegistryCreator. The access to the registry is atomically synchronised. If an error occurs, then the
// previously populated registry is used.
//
// Syncer can be used to periodically synchronise the RCM with the given RCP.
package rcm
