package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/docbiner/docbiner/internal/auth"
	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockUserStore struct {
	users map[string]*domain.User
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserStore) Create(_ context.Context, email, passwordHash, username, displayName string) (*domain.User, error) {
	if _, exists := m.users[email]; exists {
		return nil, fmt.Errorf("duplicate key: unique constraint violated")
	}
	u := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		Username:     username,
		DisplayName:  displayName,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.users[email] = u
	return u, nil
}

func (m *mockUserStore) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return u, nil
}

type mockOrgStore struct {
	orgs    []*domain.Organization
	members []*domain.OrgMember
}

func newMockOrgStore() *mockOrgStore {
	return &mockOrgStore{}
}

func (m *mockOrgStore) Create(_ context.Context, name, slug string, planID uuid.UUID) (*domain.Organization, error) {
	org := &domain.Organization{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		PlanID:    planID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.orgs = append(m.orgs, org)
	return org, nil
}

func (m *mockOrgStore) AddMember(_ context.Context, orgID, userID uuid.UUID, role domain.OrgRole, invitedBy *uuid.UUID) (*domain.OrgMember, error) {
	member := &domain.OrgMember{
		ID:        uuid.New(),
		OrgID:     orgID,
		UserID:    userID,
		Role:      role,
		InvitedBy: invitedBy,
		CreatedAt: time.Now(),
	}
	m.members = append(m.members, member)
	return member, nil
}

type mockPlanStore struct {
	plan *domain.Plan
}

func newMockPlanStore() *mockPlanStore {
	return &mockPlanStore{
		plan: &domain.Plan{
			ID:   uuid.New(),
			Name: "free",
		},
	}
}

func (m *mockPlanStore) GetByName(_ context.Context, name string) (*domain.Plan, error) {
	if m.plan != nil && m.plan.Name == name {
		return m.plan, nil
	}
	return nil, fmt.Errorf("plan not found: %s", name)
}

type mockJWTGenerator struct {
	token string
	err   error
}

func newMockJWTGenerator() *mockJWTGenerator {
	return &mockJWTGenerator{
		token: "mock-jwt-token",
	}
}

func (m *mockJWTGenerator) Generate(_, _ uuid.UUID, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

type mockOrgMemberStore struct {
	members map[uuid.UUID]*domain.OrgMember
}

func newMockOrgMemberStore() *mockOrgMemberStore {
	return &mockOrgMemberStore{
		members: make(map[uuid.UUID]*domain.OrgMember),
	}
}

func (m *mockOrgMemberStore) GetByUserID(_ context.Context, userID uuid.UUID) (*domain.OrgMember, error) {
	member, ok := m.members[userID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return member, nil
}

// --- Helpers ---

func setupAuthHandler() (*AuthHandler, *mockUserStore, *mockOrgStore, *mockPlanStore, *mockJWTGenerator, *mockOrgMemberStore) {
	users := newMockUserStore()
	orgs := newMockOrgStore()
	plans := newMockPlanStore()
	jwtGen := newMockJWTGenerator()
	orgMembers := newMockOrgMemberStore()
	handler := NewAuthHandler(users, orgs, plans, jwtGen, orgMembers)
	return handler, users, orgs, plans, jwtGen, orgMembers
}

func postJSON(path, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// --- Register Tests ---

func TestRegister_Success(t *testing.T) {
	handler, _, _, _, _, _ := setupAuthHandler()

	body := `{
		"email": "user@example.com",
		"password": "securepass123",
		"username": "johndoe",
		"display_name": "John Doe",
		"org_name": "My Company"
	}`
	c, rec := postJSON("/v1/auth/register", body)

	err := handler.Register(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp RegisterResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	assert.Equal(t, "user@example.com", resp.User.Email)
	assert.Equal(t, "johndoe", resp.User.Username)
	assert.Equal(t, "John Doe", resp.User.DisplayName)
	assert.NotEqual(t, uuid.Nil, resp.User.ID)
	assert.Equal(t, "My Company", resp.Organization.Name)
	assert.Equal(t, "my-company", resp.Organization.Slug)
	assert.NotEqual(t, uuid.Nil, resp.Organization.ID)
	assert.Equal(t, "mock-jwt-token", resp.Token)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	handler, users, _, _, _, _ := setupAuthHandler()

	// Pre-populate a user.
	users.users["user@example.com"] = &domain.User{
		ID:    uuid.New(),
		Email: "user@example.com",
	}

	body := `{
		"email": "user@example.com",
		"password": "securepass123",
		"username": "johndoe",
		"display_name": "John Doe",
		"org_name": "My Company"
	}`
	c, rec := postJSON("/v1/auth/register", body)

	err := handler.Register(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusConflict, rec.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "conflict", resp.Error)
	assert.Equal(t, "Email already registered", resp.Message)
}

func TestRegister_WeakPassword(t *testing.T) {
	handler, _, _, _, _, _ := setupAuthHandler()

	body := `{
		"email": "user@example.com",
		"password": "short",
		"username": "johndoe",
		"display_name": "John Doe",
		"org_name": "My Company"
	}`
	c, rec := postJSON("/v1/auth/register", body)

	err := handler.Register(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "validation_error", resp.Error)
	assert.Contains(t, resp.Message, "8 characters")
}

func TestRegister_InvalidEmail(t *testing.T) {
	handler, _, _, _, _, _ := setupAuthHandler()

	body := `{
		"email": "not-an-email",
		"password": "securepass123",
		"username": "johndoe",
		"display_name": "John Doe",
		"org_name": "My Company"
	}`
	c, rec := postJSON("/v1/auth/register", body)

	err := handler.Register(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "validation_error", resp.Error)
	assert.Contains(t, resp.Message, "email")
}

func TestRegister_MissingUsername(t *testing.T) {
	handler, _, _, _, _, _ := setupAuthHandler()

	body := `{
		"email": "user@example.com",
		"password": "securepass123",
		"username": "",
		"org_name": "My Company"
	}`
	c, rec := postJSON("/v1/auth/register", body)

	err := handler.Register(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_MissingOrgName(t *testing.T) {
	handler, _, _, _, _, _ := setupAuthHandler()

	body := `{
		"email": "user@example.com",
		"password": "securepass123",
		"username": "johndoe",
		"org_name": ""
	}`
	c, rec := postJSON("/v1/auth/register", body)

	err := handler.Register(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Login Tests ---

func TestLogin_Success(t *testing.T) {
	handler, users, _, _, _, orgMembers := setupAuthHandler()

	userID := uuid.New()
	orgID := uuid.New()

	passwordHash, err := auth.HashPassword("securepass123")
	require.NoError(t, err)

	users.users["user@example.com"] = &domain.User{
		ID:           userID,
		Email:        "user@example.com",
		PasswordHash: passwordHash,
		Username:     "johndoe",
		DisplayName:  "John Doe",
	}

	orgMembers.members[userID] = &domain.OrgMember{
		ID:     uuid.New(),
		OrgID:  orgID,
		UserID: userID,
		Role:   domain.OrgRoleOwner,
	}

	body := `{
		"email": "user@example.com",
		"password": "securepass123"
	}`
	c, rec := postJSON("/v1/auth/login", body)

	err = handler.Login(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp LoginResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, userID, resp.User.ID)
	assert.Equal(t, "user@example.com", resp.User.Email)
	assert.Equal(t, "mock-jwt-token", resp.Token)
}

func TestLogin_WrongPassword(t *testing.T) {
	handler, users, _, _, _, _ := setupAuthHandler()

	passwordHash, err := auth.HashPassword("securepass123")
	require.NoError(t, err)

	users.users["user@example.com"] = &domain.User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		PasswordHash: passwordHash,
		Username:     "johndoe",
		DisplayName:  "John Doe",
	}

	body := `{
		"email": "user@example.com",
		"password": "wrongpassword"
	}`
	c, rec := postJSON("/v1/auth/login", body)

	err = handler.Login(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "unauthorized", resp.Error)
}

func TestLogin_UnknownEmail(t *testing.T) {
	handler, _, _, _, _, _ := setupAuthHandler()

	body := `{
		"email": "nobody@example.com",
		"password": "securepass123"
	}`
	c, rec := postJSON("/v1/auth/login", body)

	err := handler.Login(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "unauthorized", resp.Error)
}

func TestLogin_EmptyFields(t *testing.T) {
	handler, _, _, _, _, _ := setupAuthHandler()

	body := `{"email": "", "password": ""}`
	c, rec := postJSON("/v1/auth/login", body)

	err := handler.Login(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Slugify Tests ---

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My Company", "my-company"},
		{"  Acme Corp  ", "acme-corp"},
		{"Foo & Bar", "foo-bar"},
		{"Hello_World", "hello-world"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Special!@#$%Chars", "specialchars"},
		{"already-slugged", "already-slugged"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := slugify(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}
