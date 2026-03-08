package domain

import (
	"time"

	"github.com/google/uuid"
)

// --- Organization ---

type Organization struct {
	ID               uuid.UUID `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	Slug             string    `json:"slug" db:"slug"`
	PlanID           uuid.UUID `json:"plan_id" db:"plan_id"`
	StripeCustomerID string    `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// --- User ---

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Username     string    `json:"username" db:"username"`
	DisplayName  string    `json:"display_name" db:"display_name"`
	AvatarURL    *string   `json:"avatar_url,omitempty" db:"avatar_url"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// --- Organization Members ---

type OrgRole string

const (
	OrgRoleOwner  OrgRole = "owner"
	OrgRoleAdmin  OrgRole = "admin"
	OrgRoleMember OrgRole = "member"
)

type OrgMember struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	OrgID     uuid.UUID  `json:"org_id" db:"org_id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Role      OrgRole    `json:"role" db:"role"`
	InvitedBy *uuid.UUID `json:"invited_by,omitempty" db:"invited_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// --- API Keys ---

type APIKeyEnvironment string

const (
	APIKeyEnvLive APIKeyEnvironment = "live"
	APIKeyEnvTest APIKeyEnvironment = "test"
)

type APIKey struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	OrgID       uuid.UUID         `json:"org_id" db:"org_id"`
	CreatedBy   uuid.UUID         `json:"created_by" db:"created_by"`
	KeyHash     string            `json:"-" db:"key_hash"`
	KeyPrefix   string            `json:"key_prefix" db:"key_prefix"`
	Name        string            `json:"name" db:"name"`
	Environment APIKeyEnvironment `json:"environment" db:"environment"`
	LastUsedAt  *time.Time        `json:"last_used_at,omitempty" db:"last_used_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
}

// --- Jobs ---

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

type InputType string

const (
	InputTypeURL      InputType = "url"
	InputTypeHTML     InputType = "html"
	InputTypeTemplate InputType = "template"
)

type OutputFormat string

const (
	OutputFormatPDF  OutputFormat = "pdf"
	OutputFormatPNG  OutputFormat = "png"
	OutputFormatJPEG OutputFormat = "jpeg"
	OutputFormatWebP OutputFormat = "webp"
)

type DeliveryMethod string

const (
	DeliverySync    DeliveryMethod = "sync"
	DeliveryWebhook DeliveryMethod = "webhook"
	DeliveryS3      DeliveryMethod = "s3"
)

type Job struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	OrgID          uuid.UUID      `json:"org_id" db:"org_id"`
	APIKeyID       uuid.UUID      `json:"api_key_id" db:"api_key_id"`
	Status         JobStatus      `json:"status" db:"status"`
	InputType      InputType      `json:"input_type" db:"input_type"`
	InputSource    string         `json:"input_source" db:"input_source"`
	InputData      []byte         `json:"input_data,omitempty" db:"input_data"`
	OutputFormat   OutputFormat   `json:"output_format" db:"output_format"`
	Options        []byte         `json:"options" db:"options"`
	DeliveryMethod DeliveryMethod `json:"delivery_method" db:"delivery_method"`
	DeliveryConfig []byte         `json:"delivery_config,omitempty" db:"delivery_config"`
	ResultURL      *string        `json:"result_url,omitempty" db:"result_url"`
	ResultSize     *int64         `json:"result_size,omitempty" db:"result_size"`
	PagesCount     *int           `json:"pages_count,omitempty" db:"pages_count"`
	DurationMs     *int64         `json:"duration_ms,omitempty" db:"duration_ms"`
	ErrorMessage   *string        `json:"error_message,omitempty" db:"error_message"`
	IsTest         bool           `json:"is_test" db:"is_test"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty" db:"completed_at"`
}

// --- Templates ---

type TemplateEngine string

const (
	TemplateEngineHandlebars TemplateEngine = "handlebars"
	TemplateEngineLiquid     TemplateEngine = "liquid"
)

type Template struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	OrgID       uuid.UUID      `json:"org_id" db:"org_id"`
	CreatedBy   uuid.UUID      `json:"created_by" db:"created_by"`
	Name        string         `json:"name" db:"name"`
	Engine      TemplateEngine `json:"engine" db:"engine"`
	HTMLContent string         `json:"html_content" db:"html_content"`
	CSSContent  *string        `json:"css_content,omitempty" db:"css_content"`
	SampleData  []byte         `json:"sample_data,omitempty" db:"sample_data"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// --- Plans ---

type Plan struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	MonthlyQuota   int       `json:"monthly_quota" db:"monthly_quota"`
	OveragePrice   float64   `json:"overage_price" db:"overage_price"`
	PriceMonthly   float64   `json:"price_monthly" db:"price_monthly"`
	PriceYearly    float64   `json:"price_yearly" db:"price_yearly"`
	MaxFileSize    int       `json:"max_file_size" db:"max_file_size"`
	TimeoutSeconds int       `json:"timeout_seconds" db:"timeout_seconds"`
	Features       []byte    `json:"features" db:"features"`
	Active         bool      `json:"active" db:"active"`
}

// --- Usage ---

type UsageMonthly struct {
	ID              uuid.UUID `json:"id" db:"id"`
	OrgID           uuid.UUID `json:"org_id" db:"org_id"`
	Month           time.Time `json:"month" db:"month"`
	Conversions     int       `json:"conversions" db:"conversions"`
	TestConversions int       `json:"test_conversions" db:"test_conversions"`
	OverageAmount   float64   `json:"overage_amount" db:"overage_amount"`
}
