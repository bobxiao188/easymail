package extractors

import (
	"context"
	"strings"
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/easydns"

	spflib "blitiri.com.ar/go/spf"
)

type spfExtractor struct{}

func (spfExtractor) Key() string         { return "spf_check" }
func (spfExtractor) Stage() filter.Stage { return filter.StageMailFrom }

func (spfExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil || fc.ConnectIP == nil {
		return spfSkipped("no_connect_ip"), nil
	}

	// Validate IP address format
	if ip := fc.ConnectIP.To4(); ip == nil && fc.ConnectIP.To16() == nil {
		return filter.FeatureBatch{
			"spf_result": spfCodePermError,
			"spf_error":  1,
		}, nil
	}

	sender := strings.TrimSpace(fc.MailFrom)
	if sender == "" {
		return spfSkipped("no_sender"), nil
	}

	domain := extractDomainFromAddr(sender)
	if domain == "" {
		return spfSkipped("no_domain"), nil
	}

	// Validate domain length (RFC 5321: max 255 characters)
	if len(domain) > 255 {
		return filter.FeatureBatch{
			"spf_result": spfCodePermError,
			"spf_error":  1,
		}, nil
	}

	// Set context timeout for SPF check (30 seconds)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resolver := easydns.GetDefault()
	if resolver == nil {
		return spfSkipped("no_resolver"), nil
	}

	helo := strings.TrimSpace(fc.HeloName)
	if helo == "" {
		helo = "localhost"
	}

	result, err := spflib.CheckHostWithSender(
		fc.ConnectIP, helo, sender,
		spflib.WithContext(ctx),
		spflib.WithResolver(resolver),
		spflib.OverrideLookupLimit(25),
		spflib.OverrideVoidLookupLimit(25),
	)

	if err != nil {
		return filter.FeatureBatch{
			"spf_result": spfCodeTempError,
			"spf_error":  1,
		}, nil
	}

	return spfResultToFeatures(result), nil
}

func spfResultToFeatures(r spflib.Result) filter.FeatureBatch {
	code := spfCodePass
	switch r {
	case spflib.None:
		code = spfCodeNone
	case spflib.Neutral:
		code = spfCodeNeutral
	case spflib.Pass:
		code = spfCodePass
	case spflib.Fail:
		code = spfCodeFail
	case spflib.SoftFail:
		code = spfCodeSoftFail
	case spflib.TempError:
		code = spfCodeTempError
	case spflib.PermError:
		code = spfCodePermError
	}
	return filter.FeatureBatch{
		"spf_result":   code,
		"spf_pass":     boolToFloat64(r == spflib.Pass),
		"spf_fail":     boolToFloat64(r == spflib.Fail || r == spflib.SoftFail),
		"spf_softfail": boolToFloat64(r == spflib.SoftFail),
		"spf_neutral":  boolToFloat64(r == spflib.Neutral),
		"spf_none":     boolToFloat64(r == spflib.None),
		"spf_error":    boolToFloat64(r == spflib.TempError || r == spflib.PermError),
	}
}

func spfSkipped(reason string) filter.FeatureBatch {
	return filter.FeatureBatch{
		"spf_result":  spfCodeNone,
		"spf_skipped": 1,
	}
}

const (
	spfCodeNone      = 0.0
	spfCodeNeutral   = 1.0
	spfCodePass      = 2.0
	spfCodeSoftFail  = 3.0
	spfCodeFail      = 4.0
	spfCodeTempError = 5.0
	spfCodePermError = 6.0
)

func init() {
	rule.Register(spfExtractor{})
}
