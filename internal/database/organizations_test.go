package database

import (
	"context"
	"testing"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
)

func TestOrgRepo_Create(t *testing.T) {
	pool := testDB(t)
	orgRepo := &OrgRepo{pool: pool}
	ctx := context.Background()

	planID := freePlanID(t, pool)
	planUUID, err := uuid.Parse(planID)
	if err != nil {
		t.Fatalf("parse plan UUID: %v", err)
	}

	suffix := uuid.NewString()[:8]
	slug := "acme-corp-" + suffix
	name := "Acme Corp " + suffix

	org, err := orgRepo.Create(ctx, name, slug, planUUID)
	if err != nil {
		t.Fatalf("Create org: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", org.ID)
	})

	if org == nil {
		t.Fatal("expected org to be non-nil")
	}
	if org.Name != name {
		t.Errorf("name = %q, want %q", org.Name, name)
	}
	if org.Slug != slug {
		t.Errorf("slug = %q, want %q", org.Slug, slug)
	}
	if org.PlanID != planUUID {
		t.Errorf("plan_id = %v, want %v", org.PlanID, planUUID)
	}
	if org.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestOrgRepo_GetBySlug(t *testing.T) {
	pool := testDB(t)
	orgRepo := &OrgRepo{pool: pool}
	ctx := context.Background()

	planID := freePlanID(t, pool)
	planUUID, _ := uuid.Parse(planID)

	suffix := uuid.NewString()[:8]
	slug := "slug-test-" + suffix

	created, err := orgRepo.Create(ctx, "Slug Test Org "+suffix, slug, planUUID)
	if err != nil {
		t.Fatalf("Create org: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", created.ID)
	})

	found, err := orgRepo.GetBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if found == nil {
		t.Fatal("expected org to be non-nil")
	}
	if found.ID != created.ID {
		t.Errorf("ID = %v, want %v", found.ID, created.ID)
	}
}

func TestOrgRepo_AddMember(t *testing.T) {
	pool := testDB(t)
	orgRepo := &OrgRepo{pool: pool}
	userRepo := &UserRepo{pool: pool}
	ctx := context.Background()

	suffix := uuid.NewString()[:8]

	// Create a user.
	user, err := userRepo.Create(ctx, "member-"+suffix+"@example.com", "pw_hash", "memberuser"+suffix, "Member User")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})

	// Create an org.
	planID := freePlanID(t, pool)
	planUUID, _ := uuid.Parse(planID)
	org, err := orgRepo.Create(ctx, "Member Test Org "+suffix, "member-test-"+suffix, planUUID)
	if err != nil {
		t.Fatalf("Create org: %v", err)
	}
	t.Cleanup(func() {
		// org_members cascade-delete with org
		_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", org.ID)
	})

	// Add member.
	member, err := orgRepo.AddMember(ctx, org.ID, user.ID, domain.OrgRoleOwner, nil)
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if member == nil {
		t.Fatal("expected member to be non-nil")
	}
	if member.OrgID != org.ID {
		t.Errorf("org_id = %v, want %v", member.OrgID, org.ID)
	}
	if member.UserID != user.ID {
		t.Errorf("user_id = %v, want %v", member.UserID, user.ID)
	}
	if member.Role != domain.OrgRoleOwner {
		t.Errorf("role = %q, want %q", member.Role, domain.OrgRoleOwner)
	}
	if member.InvitedBy != nil {
		t.Errorf("invited_by = %v, want nil", member.InvitedBy)
	}
}

func TestOrgRepo_AddMember_WithInvitedBy(t *testing.T) {
	pool := testDB(t)
	orgRepo := &OrgRepo{pool: pool}
	userRepo := &UserRepo{pool: pool}
	ctx := context.Background()

	suffix := uuid.NewString()[:8]

	// Create two users.
	owner, err := userRepo.Create(ctx, "owner-"+suffix+"@example.com", "pw", "owneruser"+suffix, "Owner")
	if err != nil {
		t.Fatalf("Create owner: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", owner.ID)
	})

	invitee, err := userRepo.Create(ctx, "invitee-"+suffix+"@example.com", "pw", "invitee"+suffix, "Invitee")
	if err != nil {
		t.Fatalf("Create invitee: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", invitee.ID)
	})

	// Create org and add owner.
	planID := freePlanID(t, pool)
	planUUID, _ := uuid.Parse(planID)
	org, err := orgRepo.Create(ctx, "Invite Test Org "+suffix, "invite-test-"+suffix, planUUID)
	if err != nil {
		t.Fatalf("Create org: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", org.ID)
	})

	_, err = orgRepo.AddMember(ctx, org.ID, owner.ID, domain.OrgRoleOwner, nil)
	if err != nil {
		t.Fatalf("AddMember (owner): %v", err)
	}

	// Add invitee with invitedBy.
	member, err := orgRepo.AddMember(ctx, org.ID, invitee.ID, domain.OrgRoleMember, &owner.ID)
	if err != nil {
		t.Fatalf("AddMember (invitee): %v", err)
	}
	if member.InvitedBy == nil {
		t.Fatal("expected invited_by to be set")
	}
	if *member.InvitedBy != owner.ID {
		t.Errorf("invited_by = %v, want %v", *member.InvitedBy, owner.ID)
	}
}
