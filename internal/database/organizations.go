package database

import (
	"context"
	"fmt"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrgRepo handles organization persistence.
type OrgRepo struct {
	pool *pgxpool.Pool
}

// scanOrg scans a row into a domain.Organization, handling nullable columns.
func scanOrg(scanner interface {
	Scan(dest ...any) error
}) (*domain.Organization, error) {
	var o domain.Organization
	var stripeID *string
	err := scanner.Scan(&o.ID, &o.Name, &o.Slug, &o.PlanID, &stripeID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if stripeID != nil {
		o.StripeCustomerID = *stripeID
	}
	return &o, nil
}

// Create inserts a new organization and returns it.
func (r *OrgRepo) Create(ctx context.Context, name, slug string, planID uuid.UUID) (*domain.Organization, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO organizations (name, slug, plan_id)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, slug, plan_id, stripe_customer_id, created_at, updated_at`,
		name, slug, planID,
	)
	o, err := scanOrg(row)
	if err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}
	return o, nil
}

// AddMember adds a user to an organization with the given role.
func (r *OrgRepo) AddMember(ctx context.Context, orgID, userID uuid.UUID, role domain.OrgRole, invitedBy *uuid.UUID) (*domain.OrgMember, error) {
	var m domain.OrgMember
	err := r.pool.QueryRow(ctx,
		`INSERT INTO org_members (org_id, user_id, role, invited_by)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, org_id, user_id, role, invited_by, created_at`,
		orgID, userID, role, invitedBy,
	).Scan(&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.InvitedBy, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("add org member: %w", err)
	}
	return &m, nil
}

// GetBySlug retrieves an organization by its slug.
func (r *OrgRepo) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, plan_id, stripe_customer_id, created_at, updated_at
		 FROM organizations WHERE slug = $1`,
		slug,
	)
	o, err := scanOrg(row)
	if err != nil {
		return nil, fmt.Errorf("get organization by slug: %w", err)
	}
	return o, nil
}

// GetByID retrieves an organization by its ID.
func (r *OrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, plan_id, stripe_customer_id, created_at, updated_at
		 FROM organizations WHERE id = $1`,
		id,
	)
	o, err := scanOrg(row)
	if err != nil {
		return nil, fmt.Errorf("get organization by id: %w", err)
	}
	return o, nil
}

// GetByStripeCustomerID retrieves an organization by its Stripe customer ID.
func (r *OrgRepo) GetByStripeCustomerID(ctx context.Context, customerID string) (*domain.Organization, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, plan_id, stripe_customer_id, created_at, updated_at
		 FROM organizations WHERE stripe_customer_id = $1`,
		customerID,
	)
	o, err := scanOrg(row)
	if err != nil {
		return nil, fmt.Errorf("get organization by stripe customer id: %w", err)
	}
	return o, nil
}

// UpdateStripeCustomerID sets the Stripe customer ID for an organization.
func (r *OrgRepo) UpdateStripeCustomerID(ctx context.Context, orgID uuid.UUID, customerID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE organizations SET stripe_customer_id = $2, updated_at = NOW() WHERE id = $1`,
		orgID, customerID,
	)
	if err != nil {
		return fmt.Errorf("update stripe customer id: %w", err)
	}
	return nil
}

// UpdatePlan updates the plan for an organization.
func (r *OrgRepo) UpdatePlan(ctx context.Context, orgID, planID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE organizations SET plan_id = $2, updated_at = NOW() WHERE id = $1`,
		orgID, planID,
	)
	if err != nil {
		return fmt.Errorf("update organization plan: %w", err)
	}
	return nil
}

// MemberWithUser represents an org member joined with user details.
type MemberWithUser struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	DisplayName string          `json:"display_name"`
	Email       string          `json:"email"`
	Role        domain.OrgRole  `json:"role"`
	AvatarURL   *string         `json:"avatar_url,omitempty"`
	JoinedAt    string          `json:"joined_at"`
}

// ListMembers returns all members of an organization with user details.
func (r *OrgRepo) ListMembers(ctx context.Context, orgID uuid.UUID) ([]MemberWithUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT om.id, om.user_id, u.display_name, u.email, om.role, u.avatar_url, om.created_at
		 FROM org_members om
		 JOIN users u ON u.id = om.user_id
		 WHERE om.org_id = $1
		 ORDER BY om.created_at ASC`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("list org members: %w", err)
	}
	defer rows.Close()

	var members []MemberWithUser
	for rows.Next() {
		var m MemberWithUser
		var createdAt interface{}
		if err := rows.Scan(&m.ID, &m.UserID, &m.DisplayName, &m.Email, &m.Role, &m.AvatarURL, &createdAt); err != nil {
			return nil, fmt.Errorf("scan org member: %w", err)
		}
		if t, ok := createdAt.(interface{ Format(string) string }); ok {
			m.JoinedAt = t.Format("2006-01-02T15:04:05Z07:00")
		}
		members = append(members, m)
	}
	if members == nil {
		members = []MemberWithUser{}
	}
	return members, nil
}

// GetMemberByUserID retrieves the first org membership for a given user.
func (r *OrgRepo) GetMemberByUserID(ctx context.Context, userID uuid.UUID) (*domain.OrgMember, error) {
	var m domain.OrgMember
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, user_id, role, invited_by, created_at
		 FROM org_members WHERE user_id = $1
		 ORDER BY created_at ASC LIMIT 1`,
		userID,
	).Scan(&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.InvitedBy, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get org member by user id: %w", err)
	}
	return &m, nil
}
