package game

import "strings"

// standardAttributeKeys is the canonical ordered set of base attributes.
var standardAttributeKeys = []string{
	"strength", "dexterity", "constitution", "intelligence", "wisdom", "charisma",
}

func isStandardAttributeKey(key string) bool {
	for _, k := range standardAttributeKeys {
		if k == key {
			return true
		}
	}
	return false
}

// parseEnabledStandardAttrs turns the stored CSV into a validated, canonically
// ordered slice of enabled attribute keys (unknown/duplicate entries dropped).
func parseEnabledStandardAttrs(csv string) []string {
	present := make(map[string]bool)
	for _, part := range strings.Split(csv, ",") {
		key := strings.ToLower(strings.TrimSpace(part))
		if isStandardAttributeKey(key) {
			present[key] = true
		}
	}

	enabled := make([]string, 0, len(standardAttributeKeys))
	for _, key := range standardAttributeKeys {
		if present[key] {
			enabled = append(enabled, key)
		}
	}
	return enabled
}

// formatEnabledStandardAttrs normalizes an incoming list (e.g. from a request)
// into the canonical CSV stored on the game.
func formatEnabledStandardAttrs(keys []string) string {
	requested := make(map[string]bool, len(keys))
	for _, key := range keys {
		requested[strings.ToLower(strings.TrimSpace(key))] = true
	}

	ordered := make([]string, 0, len(standardAttributeKeys))
	for _, key := range standardAttributeKeys {
		if requested[key] {
			ordered = append(ordered, key)
		}
	}
	return strings.Join(ordered, ",")
}
