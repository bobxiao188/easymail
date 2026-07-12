// Package milter implements the MTA protocol adapter for the filter.
//
//	handler.go        — MilterHandler wiring, factories, logging helpers
//	handler_stages.go — protocol/milter callbacks (Connect → Body)
//	handler_eval.go   — feature eval, policy response, async filter_logs + Redis intraday
//
// Policy uses easymail/internal/app/filter; features use easymail/internal/domain/filter/feature.
package milter
