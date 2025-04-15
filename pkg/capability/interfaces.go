// Package capability defines the capabilities for the MCP protocol
package capability

import (
	"context"
)

// ClientCapability rappresenta una capability implementata dal client
type ClientCapability interface {
	Capability
	// IsClientCapability identifica questa come una capability client
	IsClientCapability() bool
}

// ServerCapability rappresenta una capability implementata dal server
type ServerCapability interface {
	Capability
	// IsServerCapability identifica questa come una capability server
	IsServerCapability() bool
}

// ListChangeNotifier rappresenta una capability che supporta notifiche di cambiamento della lista
type ListChangeNotifier interface {
	// SupportsListChangedNotifications indica se la capability supporta notifiche di cambiamento della lista
	SupportsListChangedNotifications() bool
	// NotifyListChanged invia una notifica che la lista è cambiata
	NotifyListChanged(ctx context.Context) error
}

// SubscriptionSupporter rappresenta una capability che supporta la sottoscrizione a elementi individuali
type SubscriptionSupporter interface {
	// SupportsSubscription indica se la capability supporta la sottoscrizione
	SupportsSubscription() bool
}
