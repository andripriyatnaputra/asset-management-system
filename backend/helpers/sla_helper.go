package helpers

import "strings"

// ============================================================
// 🧮 SLA Compliance Score Helpers (Grade A ++)
// ============================================================

// V2 - Dynamic Score (aktif)
func ComputeDynamicScore(response, resolve int, impact, urgency string) float64 {
	base := 100.0
	impact = strings.ToLower(strings.TrimSpace(impact))
	urgency = strings.ToLower(strings.TrimSpace(urgency))

	// Adjust base by impact–urgency
	switch impact {
	case "high":
		base -= 10
	case "medium":
		base -= 5
	}

	switch urgency {
	case "high":
		base -= 5
	case "low":
		base += 5
	}

	// Penalize slow response or resolution
	if response > 60 {
		base -= float64(response-60) * 0.1
	}
	if resolve > 240 {
		base -= float64(resolve-240) * 0.05
	}
	if base < 0 {
		base = 0
	}
	return base
}

// V1 - Legacy Score (optional for backward analytics)
func ComputeLegacyComplianceScore(impact, urgency, priority string) float64 {
	impact = strings.ToLower(impact)
	urgency = strings.ToLower(urgency)
	priority = strings.ToLower(priority)

	expected := map[string]map[string]string{
		"low": {
			"low":    "low",
			"medium": "low",
			"high":   "medium",
		},
		"medium": {
			"low":    "low",
			"medium": "medium",
			"high":   "high",
		},
		"high": {
			"low":    "medium",
			"medium": "high",
			"high":   "critical",
		},
	}
	exp := expected[impact][urgency]
	if exp == priority {
		return 100
	}
	return 70
}
