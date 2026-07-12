package extractors

import (
	"context"
	"strings"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
)

// rcptLocalExtractor checks if recipients/domains are local and valid.
// Note: DB-dependent cache checks moved to app layer.
type rcptLocalExtractor struct{}

func (rcptLocalExtractor) Key() string         { return "rcpt_local" }
func (rcptLocalExtractor) Stage() filter.Stage { return filter.StageRcptTo }
func (rcptLocalExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil || len(fc.Rcpts) == 0 {
		return nil, nil
	}
	rcpt := strings.ToLower(strings.TrimSpace(fc.Rcpts[len(fc.Rcpts)-1]))
	if rcpt == "" {
		return nil, nil
	}
	// Local domain/mailbox checks require DB access; handled by app-layer extractor.
	return filter.FeatureBatch{
		"rcpt_domain_is_local": 0,
		"rcpt_mailbox_exists":  0,
	}, nil
}

func init() {
	rule.Register(rcptLocalExtractor{})
}

func init() {
	rule.Register(rcptLocalExtractor{})
}
