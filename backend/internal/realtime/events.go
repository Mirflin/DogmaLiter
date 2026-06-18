package realtime

// Event type identifiers pushed to clients. Each one is an invalidation signal:
// the client refreshes the matching slice of state via the existing REST API.
const (
	// EventCharactersChanged: a character was created or deleted in the game —
	// clients should reload the session character list (fixes the GM roster
	// not updating when a new character appears).
	EventCharactersChanged = "characters_changed"

	// EventCharacterUpdated: a single character changed (stats, inventory,
	// equipment, currency). Data carries "character_id".
	EventCharacterUpdated = "character_updated"

	// EventTradesChanged: a trade offer was created or resolved — clients
	// should reload their incoming/outgoing trades.
	EventTradesChanged = "trades_changed"

	// EventChatMessage: a new chat message was posted.
	EventChatMessage = "chat_message"

	// EventActivityChanged: the activity log changed (GM manage panel).
	EventActivityChanged = "activity_changed"
)
