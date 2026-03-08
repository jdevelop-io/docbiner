package handlers

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/docbiner/docbiner/internal/auth"
	"github.com/docbiner/docbiner/internal/database"
	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// --- Interfaces ---

// UserStore abstracts user persistence for auth handlers.
type UserStore interface {
	Create(ctx context.Context, email, passwordHash, username, displayName string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// OrgStore abstracts organization persistence for auth handlers.
type OrgStore interface {
	Create(ctx context.Context, name, slug string, planID uuid.UUID) (*domain.Organization, error)
	AddMember(ctx context.Context, orgID, userID uuid.UUID, role domain.OrgRole, invitedBy *uuid.UUID) (*domain.OrgMember, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
}

// PlanStore abstracts plan lookups for auth handlers.
type PlanStore interface {
	GetByName(ctx context.Context, name string) (*domain.Plan, error)
}

// OrgMemberStore abstracts org membership lookups for login.
type OrgMemberStore interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.OrgMember, error)
	ListMembers(ctx context.Context, orgID uuid.UUID) ([]database.MemberWithUser, error)
}

// JWTGenerator abstracts JWT token generation.
type JWTGenerator interface {
	Generate(userID, orgID uuid.UUID, role string) (string, error)
}

// --- Request/Response structs ---

// RegisterRequest is the JSON body for POST /v1/auth/register.
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	OrgName     string `json:"org_name"`
}

// LoginRequest is the JSON body for POST /v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthUserResponse is the user portion of auth responses.
type AuthUserResponse struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
}

// AuthOrgResponse is the organization portion of the register response.
type AuthOrgResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

// RegisterResponse is the JSON body returned on successful registration.
type RegisterResponse struct {
	User         AuthUserResponse `json:"user"`
	Organization AuthOrgResponse  `json:"organization"`
	Token        string           `json:"token"`
}

// LoginResponse is the JSON body returned on successful login.
type LoginResponse struct {
	User  AuthUserResponse `json:"user"`
	Token string           `json:"token"`
}

// --- Handler ---

// AuthHandler handles registration and login endpoints.
type AuthHandler struct {
	users         UserStore
	orgs          OrgStore
	plans         PlanStore
	jwt           JWTGenerator
	orgMembership OrgMemberStore
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(users UserStore, orgs OrgStore, plans PlanStore, jwt JWTGenerator, orgMembership OrgMemberStore) *AuthHandler {
	return &AuthHandler{
		users:         users,
		orgs:          orgs,
		plans:         plans,
		jwt:           jwt,
		orgMembership: orgMembership,
	}
}

// emailRegex is a simple email validation pattern.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Register handles POST /v1/auth/register.
func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	// Validate input.
	if !emailRegex.MatchString(req.Email) {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid email format",
		})
	}
	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Password must be at least 8 characters",
		})
	}
	if strings.TrimSpace(req.Username) == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Username is required",
		})
	}
	if strings.TrimSpace(req.OrgName) == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Organization name is required",
		})
	}

	ctx := c.Request().Context()

	// Check if email is already taken.
	existing, _ := h.users.GetByEmail(ctx, req.Email)
	if existing != nil {
		return c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "conflict",
			Message: "Email already registered",
		})
	}

	// Hash password.
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to hash password",
		})
	}

	// Create user.
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}

	user, err := h.users.Create(ctx, req.Email, hash, req.Username, displayName)
	if err != nil {
		// Handle unique constraint violation (duplicate email race condition).
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "conflict",
				Message: "Email already registered",
			})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create user",
		})
	}

	// Look up the free plan.
	plan, err := h.plans.GetByName(ctx, "free")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to look up free plan",
		})
	}

	// Create organization.
	slug := slugify(req.OrgName)
	org, err := h.orgs.Create(ctx, req.OrgName, slug, plan.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create organization",
		})
	}

	// Add user as org owner.
	_, err = h.orgs.AddMember(ctx, org.ID, user.ID, domain.OrgRoleOwner, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to add user to organization",
		})
	}

	// Generate JWT.
	token, err := h.jwt.Generate(user.ID, org.ID, string(domain.OrgRoleOwner))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate token",
		})
	}

	return c.JSON(http.StatusCreated, RegisterResponse{
		User: AuthUserResponse{
			ID:          user.ID,
			Email:       user.Email,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		},
		Organization: AuthOrgResponse{
			ID:   org.ID,
			Name: org.Name,
			Slug: org.Slug,
		},
		Token: token,
	})
}

// Login handles POST /v1/auth/login.
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Email and password are required",
		})
	}

	ctx := c.Request().Context()

	// Look up user.
	user, err := h.users.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid email or password",
		})
	}

	// Check password.
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid email or password",
		})
	}

	orgID := uuid.Nil
	role := string(domain.OrgRoleMember)

	if h.orgMembership != nil {
		member, err := h.orgMembership.GetByUserID(ctx, user.ID)
		if err == nil && member != nil {
			orgID = member.OrgID
			role = string(member.Role)
		}
	}

	token, err := h.jwt.Generate(user.ID, orgID, role)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate token",
		})
	}

	return c.JSON(http.StatusOK, LoginResponse{
		User: AuthUserResponse{
			ID:          user.ID,
			Email:       user.Email,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		},
		Token: token,
	})
}

// Me handles GET /v1/auth/me — returns the current authenticated user.
func (h *AuthHandler) Me(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing user context",
		})
	}

	user, err := h.users.GetByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "User not found",
		})
	}

	return c.JSON(http.StatusOK, AuthUserResponse{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
	})
}

// Organization handles GET /v1/organization — returns the current org.
func (h *AuthHandler) Organization(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing organization context",
		})
	}

	org, err := h.orgs.GetByID(c.Request().Context(), orgID)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Organization not found",
		})
	}

	return c.JSON(http.StatusOK, AuthOrgResponse{
		ID:   org.ID,
		Name: org.Name,
		Slug: org.Slug,
	})
}

// Members handles GET /v1/organization/members — returns all org members.
func (h *AuthHandler) Members(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing organization context",
		})
	}

	members, err := h.orgMembership.ListMembers(c.Request().Context(), orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to load members",
		})
	}

	return c.JSON(http.StatusOK, members)
}

// slugify converts a name into a URL-friendly slug.
func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	// Replace spaces and underscores with hyphens.
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Remove non-alphanumeric characters except hyphens.
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	// Collapse multiple hyphens.
	result := b.String()
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return strings.Trim(result, "-")
}
