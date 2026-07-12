package filter

import "strings"

// Outcome is the final disposition for a message.
type Outcome string

const (
	OutcomeAccept     Outcome = "accept"
	OutcomeReject     Outcome = "reject"
	OutcomeSpam       Outcome = "spam"
	OutcomeQuarantine Outcome = "quarantine"
)

func NormalizeOutcome(a string) Outcome {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "reject":
		return OutcomeReject
	case "spam":
		return OutcomeSpam
	case "quarantine":
		return OutcomeQuarantine
	default:
		return OutcomeAccept
	}
}