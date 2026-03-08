package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/docbiner/docbiner/internal/delivery"
	"github.com/docbiner/docbiner/internal/domain"
)

// deliveryDispatcher routes job results to the appropriate delivery target
// based on the job's delivery method and configuration.
type deliveryDispatcher struct {
	webhook *delivery.WebhookSender
	s3      *delivery.S3Deliverer
}

// newDeliveryDispatcher creates a new deliveryDispatcher with the given webhook and S3 deliverers.
func newDeliveryDispatcher(webhook *delivery.WebhookSender, s3 *delivery.S3Deliverer) *deliveryDispatcher {
	return &deliveryDispatcher{
		webhook: webhook,
		s3:      s3,
	}
}

// Dispatch delivers job results according to the job's delivery method.
func (d *deliveryDispatcher) Dispatch(ctx context.Context, job *domain.Job, resultData []byte) error {
	switch job.DeliveryMethod {
	case domain.DeliveryWebhook:
		return d.dispatchWebhook(ctx, job)
	case domain.DeliveryS3:
		return d.dispatchS3(ctx, job, resultData)
	default:
		return fmt.Errorf("unsupported delivery method: %s", job.DeliveryMethod)
	}
}

// dispatchWebhook parses webhook config from the job and sends a notification.
func (d *deliveryDispatcher) dispatchWebhook(ctx context.Context, job *domain.Job) error {
	var cfg delivery.WebhookConfig
	if err := json.Unmarshal(job.DeliveryConfig, &cfg); err != nil {
		return fmt.Errorf("parse webhook config: %w", err)
	}

	var resultURL string
	if job.ResultURL != nil {
		resultURL = *job.ResultURL
	}

	var resultSize int64
	if job.ResultSize != nil {
		resultSize = *job.ResultSize
	}

	var pagesCount int
	if job.PagesCount != nil {
		pagesCount = *job.PagesCount
	}

	var durationMs int64
	if job.DurationMs != nil {
		durationMs = *job.DurationMs
	}

	var completedAt time.Time
	if job.CompletedAt != nil {
		completedAt = *job.CompletedAt
	}

	payload := delivery.WebhookPayload{
		JobID:       job.ID,
		Status:      string(job.Status),
		Format:      string(job.OutputFormat),
		ResultURL:   resultURL,
		ResultSize:  resultSize,
		PagesCount:  pagesCount,
		DurationMs:  durationMs,
		CreatedAt:   job.CreatedAt,
		CompletedAt: completedAt,
	}

	return d.webhook.Send(ctx, cfg, payload)
}

// dispatchS3 parses S3 config from the job and uploads the result data.
func (d *deliveryDispatcher) dispatchS3(ctx context.Context, job *domain.Job, resultData []byte) error {
	var cfg delivery.S3Config
	if err := json.Unmarshal(job.DeliveryConfig, &cfg); err != nil {
		return fmt.Errorf("parse s3 config: %w", err)
	}

	contentType := contentTypeForFormat(job.OutputFormat)
	key := fmt.Sprintf("%s/result.%s", job.ID, job.OutputFormat)

	_, err := d.s3.Deliver(ctx, cfg, key, resultData, contentType)
	return err
}
