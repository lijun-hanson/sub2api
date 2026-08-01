//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// capTestQuotaUpdater is a no-op APIKeyQuotaUpdater used to satisfy the non-nil
// guard in shouldDeductAPIKeyQuota.
type capTestQuotaUpdater struct{}

func (capTestQuotaUpdater) UpdateQuotaUsed(context.Context, int64, float64) error      { return nil }
func (capTestQuotaUpdater) UpdateRateLimitUsage(context.Context, int64, float64) error { return nil }

func TestAPIKeyEffectiveQuota(t *testing.T) {
	// Explicit key quota always wins over the default cap.
	require.Equal(t, 25.0, (&APIKey{Quota: 25}).EffectiveQuota(50))
	// No explicit quota -> the default cap applies.
	require.Equal(t, 50.0, (&APIKey{Quota: 0}).EffectiveQuota(50))
	// Neither set -> unlimited (0).
	require.Equal(t, 0.0, (&APIKey{}).EffectiveQuota(0))
}

func TestAPIKeyIsQuotaExhaustedWithDefault(t *testing.T) {
	// A key with no explicit quota is bounded by the configured default cap.
	k := &APIKey{Quota: 0, QuotaUsed: 49.99}
	require.False(t, k.IsQuotaExhaustedWithDefault(50))
	k.QuotaUsed = 50
	require.True(t, k.IsQuotaExhaustedWithDefault(50))
	k.QuotaUsed = 60
	require.True(t, k.IsQuotaExhaustedWithDefault(50))

	// Default cap 0 preserves the legacy "unset quota == unlimited" behavior.
	require.False(t, (&APIKey{Quota: 0, QuotaUsed: 1e9}).IsQuotaExhaustedWithDefault(0))

	// An explicit key quota takes precedence over (and is not raised by) the default cap.
	require.True(t, (&APIKey{Quota: 10, QuotaUsed: 10}).IsQuotaExhaustedWithDefault(1000))
	require.False(t, (&APIKey{Quota: 100, QuotaUsed: 10}).IsQuotaExhaustedWithDefault(5))

	// The legacy wrapper is exactly WithDefault(0).
	require.False(t, (&APIKey{Quota: 0, QuotaUsed: 1e9}).IsQuotaExhausted())
	require.True(t, (&APIKey{Quota: 5, QuotaUsed: 5}).IsQuotaExhausted())
}

func TestShouldDeductAPIKeyQuotaHonorsDefaultCap(t *testing.T) {
	newParams := func(defaultCap, keyQuota float64) *postUsageBillingParams {
		return &postUsageBillingParams{
			Cost:               &CostBreakdown{ActualCost: 0.01},
			APIKey:             &APIKey{Quota: keyQuota},
			APIKeyService:      capTestQuotaUpdater{},
			DefaultAPIKeyQuota: defaultCap,
		}
	}

	// No explicit key quota AND no default cap => don't track (legacy unlimited).
	require.False(t, newParams(0, 0).shouldDeductAPIKeyQuota())
	// No explicit key quota BUT a default cap => must track quota_used so the cap can trip.
	require.True(t, newParams(50, 0).shouldDeductAPIKeyQuota())
	// Explicit key quota => track regardless of the default cap.
	require.True(t, newParams(0, 10).shouldDeductAPIKeyQuota())

	// Zero actual cost never tracks.
	p := newParams(50, 0)
	p.Cost = &CostBreakdown{ActualCost: 0}
	require.False(t, p.shouldDeductAPIKeyQuota())

	// A nil quota updater never tracks.
	p = newParams(50, 0)
	p.APIKeyService = nil
	require.False(t, p.shouldDeductAPIKeyQuota())
}
