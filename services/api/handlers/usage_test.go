package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/docbiner/docbiner/internal/usage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock UsageReader ---

type mockUsageReader struct {
	current      *usage.MonthlyUsage
	currentErr   error
	history      []*usage.MonthlyUsage
	historyErr   error
	quotaStatus  *usage.QuotaStatus
	quotaErr     error
}

func (m *mockUsageReader) GetCurrent(_ context.Context, _ uuid.UUID) (*usage.MonthlyUsage, error) {
	if m.currentErr != nil {
		return nil, m.currentErr
	}
	return m.current, nil
}

func (m *mockUsageReader) GetHistory(_ context.Context, _ uuid.UUID, _ int) ([]*usage.MonthlyUsage, error) {
	if m.historyErr != nil {
		return nil, m.historyErr
	}
	return m.history, nil
}

func (m *mockUsageReader) GetQuotaStatus(_ context.Context, _ uuid.UUID) (*usage.QuotaStatus, error) {
	if m.quotaErr != nil {
		return nil, m.quotaErr
	}
	return m.quotaStatus, nil
}

// --- Test Helpers ---

func setupUsageTest(reader UsageReader) *echo.Echo {
	e := echo.New()
	h := NewUsageHandler(reader)

	e.GET("/v1/usage", func(c echo.Context) error {
		c.Set("org_id", uuid.New())
		return h.HandleGetUsage(c)
	})
	e.GET("/v1/usage/history", func(c echo.Context) error {
		c.Set("org_id", uuid.New())
		return h.HandleGetUsageHistory(c)
	})

	return e
}

func doGet(e *echo.Echo, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// --- Tests ---

func TestGetUsage_ReturnsUsageData(t *testing.T) {
	month := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mock := &mockUsageReader{
		current: &usage.MonthlyUsage{
			Month:           month,
			Conversions:     450,
			TestConversions: 23,
			OverageAmount:   0,
		},
		quotaStatus: &usage.QuotaStatus{
			Allowed:   true,
			Used:      450,
			Limit:     2500,
			Remaining: 2050,
		},
	}
	e := setupUsageTest(mock)

	rec := doGet(e, "/v1/usage")

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp UsageResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "2024-01", resp.Month)
	assert.Equal(t, 450, resp.Conversions)
	assert.Equal(t, 23, resp.TestConversions)
	assert.Equal(t, 450, resp.Quota.Used)
	assert.Equal(t, 2500, resp.Quota.Limit)
	assert.Equal(t, 2050, resp.Quota.Remaining)
	assert.True(t, resp.Quota.Allowed)
}

func TestGetUsage_NoData_ReturnsZeroValues(t *testing.T) {
	month := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	mock := &mockUsageReader{
		current: &usage.MonthlyUsage{
			Month: month,
		},
		quotaStatus: &usage.QuotaStatus{
			Allowed:   true,
			Used:      0,
			Limit:     100,
			Remaining: 100,
		},
	}
	e := setupUsageTest(mock)

	rec := doGet(e, "/v1/usage")

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp UsageResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "2024-03", resp.Month)
	assert.Equal(t, 0, resp.Conversions)
	assert.Equal(t, 0, resp.TestConversions)
	assert.Equal(t, 0, resp.Quota.Used)
	assert.Equal(t, 100, resp.Quota.Limit)
	assert.Equal(t, 100, resp.Quota.Remaining)
}

func TestGetUsage_CurrentError_Returns500(t *testing.T) {
	mock := &mockUsageReader{
		currentErr: errors.New("db error"),
	}
	e := setupUsageTest(mock)

	rec := doGet(e, "/v1/usage")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "internal_error", errResp.Error)
}

func TestGetUsage_QuotaError_Returns500(t *testing.T) {
	mock := &mockUsageReader{
		current: &usage.MonthlyUsage{
			Month: time.Now(),
		},
		quotaErr: errors.New("db error"),
	}
	e := setupUsageTest(mock)

	rec := doGet(e, "/v1/usage")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "internal_error", errResp.Error)
}

func TestGetUsageHistory_ReturnsArray(t *testing.T) {
	mock := &mockUsageReader{
		history: []*usage.MonthlyUsage{
			{
				Month:           time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				Conversions:     300,
				TestConversions: 10,
			},
			{
				Month:           time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				Conversions:     250,
				TestConversions: 5,
			},
			{
				Month:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Conversions:     100,
				TestConversions: 2,
			},
		},
	}
	e := setupUsageTest(mock)

	rec := doGet(e, "/v1/usage/history")

	assert.Equal(t, http.StatusOK, rec.Code)

	var history []*usage.MonthlyUsage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &history))
	assert.Len(t, history, 3)
	assert.Equal(t, 300, history[0].Conversions)
	assert.Equal(t, 250, history[1].Conversions)
	assert.Equal(t, 100, history[2].Conversions)
}

func TestGetUsageHistory_NoData_ReturnsEmptyArray(t *testing.T) {
	mock := &mockUsageReader{
		history: nil,
	}
	e := setupUsageTest(mock)

	rec := doGet(e, "/v1/usage/history")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "[]\n", rec.Body.String())
}

func TestGetUsageHistory_Error_Returns500(t *testing.T) {
	mock := &mockUsageReader{
		historyErr: errors.New("db error"),
	}
	e := setupUsageTest(mock)

	rec := doGet(e, "/v1/usage/history")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "internal_error", errResp.Error)
}

func TestGetUsage_NoOrgID_Returns401(t *testing.T) {
	mock := &mockUsageReader{}
	e := echo.New()
	h := NewUsageHandler(mock)

	// Do NOT set org_id in context.
	e.GET("/v1/usage", h.HandleGetUsage)

	rec := doGet(e, "/v1/usage")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetUsageHistory_NoOrgID_Returns401(t *testing.T) {
	mock := &mockUsageReader{}
	e := echo.New()
	h := NewUsageHandler(mock)

	// Do NOT set org_id in context.
	e.GET("/v1/usage/history", h.HandleGetUsageHistory)

	rec := doGet(e, "/v1/usage/history")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
