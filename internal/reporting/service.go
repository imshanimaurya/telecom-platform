package reporting

import (
	"context"
	"errors"
	"time"

	"telecom-platform/internal/calls"
	"telecom-platform/internal/wallet"
)

var ErrInvalidRequest = errors.New("reporting: invalid request")

// Repository abstracts data access for reporting.
//
// IMPORTANT:
// - Methoden must enforce workspace filtering.
// - Implementations should query immutable sources when possible (wallet ledger, audit, call records).

type Repository interface {
	ListCalls(ctx context.Context, workspaceID string, from, to time.Time, campaignID string) ([]calls.Call, error)
	ListWalletLedger(ctx context.Context, workspaceID string, from, to time.Time, walletID string) ([]wallet.WalletLedger, error)

	// Campaign conversions will likely come from a dedicated immutable events table.
	// For now this is an optional hook.
	ListConversions(ctx context.Context, workspaceID string, from, to time.Time, campaignID string) (conversions int, err error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) CallsSummary(ctx context.Context, req CallsSummaryRequest) (CallsSummary, error) {
	if req.WorkspaceID == "" {
		return CallsSummary{}, ErrInvalidRequest
	}
	if req.Range.From.IsZero() || req.Range.To.IsZero() || !req.Range.To.After(req.Range.From) {
		return CallsSummary{}, ErrInvalidRequest
	}
	if s.repo == nil {
		return CallsSummary{}, errors.New("reporting: repository not configured")
	}

	rows, err := s.repo.ListCalls(ctx, req.WorkspaceID, req.Range.From, req.Range.To, req.CampaignID)
	if err != nil {
		return CallsSummary{}, err
	}

	out := CallsSummary{WorkspaceID: req.WorkspaceID, CampaignID: req.CampaignID}
	for _, c := range rows {
		out.TotalCalls++
		out.TotalDurationSeconds += c.DurationSeconds
		if c.RecordingURL != "" {
			out.RecordedCalls++
		}
		switch c.Status {
		case calls.CallStatusCompleted:
			out.CompletedCalls++
		case calls.CallStatusFailed:
			out.FailedCalls++
		case calls.CallStatusNoAnswer:
			out.NoAnswerCalls++
		case calls.CallStatusBusy:
			out.BusyCalls++
		case calls.CallStatusCanceled:
			out.CanceledCalls++
		case calls.CallStatusInProgress:
			out.InProgressCalls++
		case calls.CallStatusRinging, calls.CallStatusQueued:
			// not counted separately
		}
	}
	if out.TotalCalls > 0 {
		out.AverageDurationSeconds = out.TotalDurationSeconds / out.TotalCalls
	}
	return out, nil
}

func (s *Service) SpendSummary(ctx context.Context, req SpendSummaryRequest) (SpendSummary, error) {
	if req.WorkspaceID == "" {
		return SpendSummary{}, ErrInvalidRequest
	}
	if req.Range.From.IsZero() || req.Range.To.IsZero() || !req.Range.To.After(req.Range.From) {
		return SpendSummary{}, ErrInvalidRequest
	}
	if s.repo == nil {
		return SpendSummary{}, errors.New("reporting: repository not configured")
	}

	ledgers, err := s.repo.ListWalletLedger(ctx, req.WorkspaceID, req.Range.From, req.Range.To, req.WalletID)
	if err != nil {
		return SpendSummary{}, err
	}

	out := SpendSummary{WorkspaceID: req.WorkspaceID, WalletID: req.WalletID, Currency: req.Currency}
	for _, l := range ledgers {
		// currency normalization: if request specified currency, filter; else populate from first row.
		if out.Currency == "" {
			out.Currency = l.Currency
		}
		if req.Currency != "" && l.Currency != req.Currency {
			continue
		}

		if l.AmountMinor > 0 {
			out.TotalCreditMinor += l.AmountMinor
		} else {
			out.TotalDebitMinor += -l.AmountMinor
		}

		// naive categorization: admin_manual_credit external ref is an admin adjustment; others count as usage.
		if l.ExternalRef == "admin_manual_credit" {
			out.AdminAdjustMinor += l.AmountMinor
		} else {
			if l.AmountMinor < 0 {
				out.UsageDebitMinor += -l.AmountMinor
			}
		}
	}
	out.NetDeltaMinor = out.TotalCreditMinor - out.TotalDebitMinor
	if out.Currency == "" {
		out.Currency = "UNKNOWN"
	}
	return out, nil
}

func (s *Service) ConversionMetrics(ctx context.Context, req ConversionMetricsRequest) (ConversionMetrics, error) {
	if req.WorkspaceID == "" || req.CampaignID == "" {
		return ConversionMetrics{}, ErrInvalidRequest
	}
	if req.Range.From.IsZero() || req.Range.To.IsZero() || !req.Range.To.After(req.Range.From) {
		return ConversionMetrics{}, ErrInvalidRequest
	}
	if s.repo == nil {
		return ConversionMetrics{}, errors.New("reporting: repository not configured")
	}

	callsRows, err := s.repo.ListCalls(ctx, req.WorkspaceID, req.Range.From, req.Range.To, req.CampaignID)
	if err != nil {
		return ConversionMetrics{}, err
	}
	conv, err := s.repo.ListConversions(ctx, req.WorkspaceID, req.Range.From, req.Range.To, req.CampaignID)
	if err != nil {
		return ConversionMetrics{}, err
	}

	out := ConversionMetrics{WorkspaceID: req.WorkspaceID, CampaignID: req.CampaignID}
	out.CallsAttempted = len(callsRows)
	for _, c := range callsRows {
		if c.Status == calls.CallStatusCompleted {
			out.CallsConnected++
		}
	}
	out.Conversions = conv

	if out.CallsAttempted > 0 {
		out.ConnectionRate = float64(out.CallsConnected) / float64(out.CallsAttempted)
		out.ConversionRate = float64(out.Conversions) / float64(out.CallsAttempted)
	}
	return out, nil
}
