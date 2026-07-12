package extractors

import (
	"context"
	"strings"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
)

// rcptContactHitExtractor checks whether the sender is in recipient\'s addressbook.
// Note: DB-dependent check (contact lookup) has been moved to the app layer.
// This domain extractor only provides the key naming.
type rcptContactHitExtractor struct{}

func (rcptContactHitExtractor) Key() string         { return "rcpt_contact_hit" }
func (rcptContactHitExtractor) Stage() filter.Stage { return filter.StageRcptTo }
func (rcptContactHitExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil || len(fc.Rcpts) == 0 {
		return nil, nil
	}
	sender := strings.ToLower(strings.TrimSpace(fc.MailFrom))
	if sender == "" {
		return filter.FeatureBatch{"rcpt_contact_sender_hit": 0}, nil
	}
	// Contact lookup requires DB access; handled by app-layer extractor.
	return filter.FeatureBatch{"rcpt_contact_sender_hit": 0}, nil
}

func init() {
	rule.Register(rcptContactHitExtractor{})
}
