package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/docbiner/docbiner/internal/auth"
	"github.com/docbiner/docbiner/internal/billing"
	"github.com/docbiner/docbiner/internal/config"
	"github.com/docbiner/docbiner/internal/database"
	"github.com/docbiner/docbiner/internal/domain"
	"github.com/docbiner/docbiner/internal/pdfutil"
	"github.com/docbiner/docbiner/internal/queue"
	"github.com/docbiner/docbiner/internal/renderer"
	"github.com/docbiner/docbiner/internal/storage"
	"github.com/docbiner/docbiner/internal/tmpl"
	"github.com/docbiner/docbiner/internal/usage"

	"github.com/docbiner/docbiner/services/api/handlers"
	"github.com/docbiner/docbiner/services/api/middleware"
)

// ---------------------------------------------------------------------------
// Adapters – bridge between handler interfaces and concrete implementations
// ---------------------------------------------------------------------------

// jobStoreAdapter adapts *database.JobRepo to handlers.JobStore.
// The handler's Create/Complete signatures differ from the DB's.
type jobStoreAdapter struct {
	repo *database.JobRepo
}

func (a *jobStoreAdapter) Create(ctx context.Context, p handlers.JobCreateParams) (*domain.Job, error) {
	return a.repo.Create(ctx, database.CreateJobParams{
		OrgID:          p.OrgID,
		APIKeyID:       p.APIKeyID,
		InputType:      p.InputType,
		InputSource:    p.InputSource,
		OutputFormat:   p.OutputFormat,
		Options:        p.Options,
		DeliveryMethod: p.DeliveryMethod,
		DeliveryConfig: p.DeliveryConfig,
		IsTest:         p.IsTest,
	})
}

func (a *jobStoreAdapter) Complete(ctx context.Context, id uuid.UUID, resultSize int64, durationMs int64) error {
	// The DB expects resultURL, resultSize, pagesCount, durationMs.
	// For synchronous convert the result is returned inline, so we pass empty values
	// for resultURL and pagesCount.
	return a.repo.Complete(ctx, id, "", resultSize, 0, durationMs)
}

func (a *jobStoreAdapter) Fail(ctx context.Context, id uuid.UUID, errMsg string, durationMs int64) error {
	return a.repo.Fail(ctx, id, errMsg, durationMs)
}

// jobReaderAdapter adapts *database.JobRepo to handlers.JobReader.
// ListByOrg has different parameter types.
type jobReaderAdapter struct {
	repo *database.JobRepo
}

func (a *jobReaderAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *jobReaderAdapter) ListByOrg(ctx context.Context, orgID uuid.UUID, p handlers.ListParams) ([]*domain.Job, int, error) {
	return a.repo.ListByOrg(ctx, database.ListJobsParams{
		OrgID:   orgID,
		Status:  p.Status,
		Format:  p.Format,
		Page:    p.Page,
		PerPage: p.PerPage,
	})
}

// templateStoreAdapter adapts *database.TemplateRepo to handlers.TemplateStore.
// Create/Update have identically-named but distinct param types.
type templateStoreAdapter struct {
	repo *database.TemplateRepo
}

func (a *templateStoreAdapter) Create(ctx context.Context, p handlers.CreateTemplateParams) (*domain.Template, error) {
	return a.repo.Create(ctx, database.CreateTemplateParams{
		OrgID:       p.OrgID,
		CreatedBy:   p.CreatedBy,
		Name:        p.Name,
		Engine:      p.Engine,
		HTMLContent: p.HTMLContent,
		CSSContent:  p.CSSContent,
		SampleData:  p.SampleData,
	})
}

func (a *templateStoreAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *templateStoreAdapter) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Template, error) {
	return a.repo.ListByOrg(ctx, orgID)
}

func (a *templateStoreAdapter) Update(ctx context.Context, id uuid.UUID, p handlers.UpdateTemplateParams) (*domain.Template, error) {
	return a.repo.Update(ctx, id, database.UpdateTemplateParams{
		Name:        p.Name,
		Engine:      p.Engine,
		HTMLContent: p.HTMLContent,
		CSSContent:  p.CSSContent,
		SampleData:  p.SampleData,
	})
}

func (a *templateStoreAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.repo.Delete(ctx, id)
}

// orgMemberAdapter adapts *database.OrgRepo to handlers.OrgMemberStore.
type orgMemberAdapter struct {
	repo *database.OrgRepo
}

func (a *orgMemberAdapter) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.OrgMember, error) {
	return a.repo.GetMemberByUserID(ctx, userID)
}

func (a *orgMemberAdapter) ListMembers(ctx context.Context, orgID uuid.UUID) ([]database.MemberWithUser, error) {
	return a.repo.ListMembers(ctx, orgID)
}

// webhookOrgAdapter adapts *database.OrgRepo to handlers.WebhookOrgStore.
// GetByStripeCustomerID returns *handlers.OrgInfo instead of *domain.Organization.
type webhookOrgAdapter struct {
	repo *database.OrgRepo
}

func (a *webhookOrgAdapter) GetByStripeCustomerID(ctx context.Context, customerID string) (*handlers.OrgInfo, error) {
	org, err := a.repo.GetByStripeCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	return &handlers.OrgInfo{
		ID:     org.ID,
		PlanID: org.PlanID,
	}, nil
}

func (a *webhookOrgAdapter) UpdatePlan(ctx context.Context, orgID, planID uuid.UUID) error {
	return a.repo.UpdatePlan(ctx, orgID, planID)
}

// webhookPlanAdapter adapts *database.PlanRepo to handlers.WebhookPlanStore.
// GetByName returns *handlers.PlanInfo instead of *domain.Plan.
type webhookPlanAdapter struct {
	repo *database.PlanRepo
}

func (a *webhookPlanAdapter) GetByName(ctx context.Context, name string) (*handlers.PlanInfo, error) {
	plan, err := a.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return &handlers.PlanInfo{
		ID:   plan.ID,
		Name: plan.Name,
	}, nil
}

// tmplRendererAdapter wraps the tmpl.Render package-level function into the
// handlers.TemplateRenderer interface.
type tmplRendererAdapter struct{}

func (tmplRendererAdapter) Render(engine string, template string, data map[string]interface{}) (string, error) {
	return tmpl.Render(engine, template, data)
}

// pdfMergerAdapter wraps the pdfutil.Merge package-level function into the
// handlers.PDFMerger interface.
type pdfMergerAdapter struct{}

func (pdfMergerAdapter) Merge(pdfs [][]byte) ([]byte, error) {
	return pdfutil.Merge(pdfs)
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	cfg := config.Load()
	ctx := context.Background()
	logger := slog.Default()

	// ---- Database ----
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	// ---- NATS ----
	q, err := queue.New(cfg.NatsURL)
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer q.Close()

	// ---- Minio storage ----
	store, err := storage.New(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, false)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	if err := store.EnsureBucket(ctx); err != nil {
		log.Fatalf("storage ensure bucket: %v", err)
	}

	// ---- Renderer (Chromium) ----
	r, err := renderer.New()
	if err != nil {
		log.Fatalf("renderer: %v", err)
	}
	defer r.Close()

	// ---- JWT service ----
	jwtSvc := auth.New(cfg.JWTSecret, 24*time.Hour)

	// ---- Billing (Stripe) ----
	stripeSvc := billing.New(os.Getenv("STRIPE_SECRET_KEY"), os.Getenv("STRIPE_WEBHOOK_SECRET"))

	// ---- Usage tracker ----
	usageTracker := usage.New(db.Pool)

	// ---- Adapters ----
	jobStore := &jobStoreAdapter{repo: db.Jobs}
	jobReader := &jobReaderAdapter{repo: db.Jobs}
	tmplStore := &templateStoreAdapter{repo: db.Templates}
	orgMembers := &orgMemberAdapter{repo: db.Organizations}
	whOrgStore := &webhookOrgAdapter{repo: db.Organizations}
	whPlanStore := &webhookPlanAdapter{repo: db.Plans}
	tmplRenderer := tmplRendererAdapter{}
	merger := pdfMergerAdapter{}

	// ---- Echo server ----
	e := echo.New()
	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{"Authorization", "Content-Type", "Accept"},
	}))

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// ---- Public routes (no auth) ----

	// Auth
	authHandler := handlers.NewAuthHandler(db.Users, db.Organizations, db.Plans, jwtSvc, orgMembers)
	e.POST("/v1/auth/register", authHandler.Register)
	e.POST("/v1/auth/login", authHandler.Login)

	// Stripe webhook (uses signature verification, no auth middleware)
	stripeWH := handlers.NewStripeWebhookHandler(stripeSvc, whOrgStore, whPlanStore, logger)
	e.POST("/v1/webhooks/stripe", stripeWH.Handle)

	// ---- API Key auth routes ----
	apiGroup := e.Group("/v1")
	apiGroup.Use(middleware.APIKeyAuth(db.APIKeys))

	// Convert
	convertH := handlers.NewConvertHandler(r, jobStore)
	apiGroup.POST("/convert", convertH.Handle)

	asyncH := handlers.NewConvertAsyncHandler(jobStore, q)
	apiGroup.POST("/convert/async", asyncH.Handle)

	// Jobs
	jobsH := handlers.NewJobsHandler(jobReader, db.Jobs, store)
	apiGroup.GET("/jobs", jobsH.List)
	apiGroup.GET("/jobs/:id", jobsH.GetByID)
	apiGroup.GET("/jobs/:id/download", jobsH.Download)
	apiGroup.DELETE("/jobs/:id", jobsH.Delete)

	// Templates
	templatesH := handlers.NewTemplateHandler(tmplStore, tmplRenderer)
	apiGroup.POST("/templates", templatesH.Create)
	apiGroup.GET("/templates", templatesH.List)
	apiGroup.POST("/templates/preview", templatesH.PreviewInline)
	apiGroup.GET("/templates/:id", templatesH.Get)
	apiGroup.PUT("/templates/:id", templatesH.Update)
	apiGroup.DELETE("/templates/:id", templatesH.Delete)
	apiGroup.POST("/templates/:id/preview", templatesH.Preview)

	// Merge
	mergeH := handlers.NewMergeHandler(r, merger)
	apiGroup.POST("/merge", mergeH.Handle)

	// Usage
	usageH := handlers.NewUsageHandler(usageTracker)
	apiGroup.GET("/usage", usageH.HandleGetUsage)
	apiGroup.GET("/usage/history", usageH.HandleGetUsageHistory)

	// ---- JWT auth routes (dashboard) ----
	dashGroup := e.Group("/v1")
	dashGroup.Use(middleware.JWTAuth(jwtSvc))

	// Auth (JWT-protected)
	dashGroup.GET("/auth/me", authHandler.Me)
	dashGroup.GET("/organization", authHandler.Organization)
	dashGroup.GET("/organization/members", authHandler.Members)

	// API Keys (dashboard)
	apiKeysH := handlers.NewAPIKeyHandler(db.APIKeys)
	dashGroup.POST("/api-keys", apiKeysH.Create)
	dashGroup.GET("/api-keys", apiKeysH.List)
	dashGroup.DELETE("/api-keys/:id", apiKeysH.Delete)

	// Convert (dashboard playground)
	dashGroup.POST("/convert", convertH.Handle)

	// Jobs (dashboard — same handlers, JWT auth)
	dashGroup.GET("/jobs", jobsH.List)
	dashGroup.GET("/jobs/:id", jobsH.GetByID)
	dashGroup.GET("/jobs/:id/download", jobsH.Download)
	dashGroup.DELETE("/jobs/:id", jobsH.Delete)

	// Templates (dashboard)
	dashGroup.POST("/templates", templatesH.Create)
	dashGroup.GET("/templates", templatesH.List)
	dashGroup.POST("/templates/preview", templatesH.PreviewInline)
	dashGroup.GET("/templates/:id", templatesH.Get)
	dashGroup.PUT("/templates/:id", templatesH.Update)
	dashGroup.DELETE("/templates/:id", templatesH.Delete)
	dashGroup.POST("/templates/:id/preview", templatesH.Preview)

	// Usage (dashboard)
	dashGroup.GET("/usage", usageH.HandleGetUsage)
	dashGroup.GET("/usage/history", usageH.HandleGetUsageHistory)

	// Billing
	billingH := handlers.NewBillingHandler(stripeSvc, db.Organizations, db.Plans)
	dashGroup.POST("/billing/checkout", billingH.HandleCheckout)
	dashGroup.POST("/billing/portal", billingH.HandlePortal)
	dashGroup.GET("/billing/status", billingH.HandleStatus)

	// ---- Graceful shutdown ----
	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	shutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutCtx); err != nil {
		log.Fatal(err)
	}
}
