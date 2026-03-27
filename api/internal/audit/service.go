package audit

import (
	"context"

	"github.com/google/uuid"

	"github.com/dokkiitech/ashiato/api/internal/domain"
	"github.com/dokkiitech/ashiato/api/internal/repository"
)

func WriteUser(
	ctx context.Context,
	store *repository.Store,
	actor domain.Actor,
	ip string,
	action string,
	resourceType string,
	resourceID uuid.UUID,
	before any,
	after any,
) error {
	actorUserID := actor.UserID.String()
	return store.CreateAuditLog(ctx, repository.AuditLogDoc{
		OrganizationID: actor.OrganizationID.String(),
		ActorUserID:    &actorUserID,
		ActorType:      "user",
		ActorLabel:     actor.Email,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID.String(),
		BeforeState:    before,
		AfterState:     after,
		Result:         "success",
		IP:             ip,
	})
}

func WriteService(
	ctx context.Context,
	store *repository.Store,
	organizationID uuid.UUID,
	label string,
	ip string,
	action string,
	resourceType string,
	resourceID uuid.UUID,
	before any,
	after any,
) error {
	return store.CreateAuditLog(ctx, repository.AuditLogDoc{
		OrganizationID: organizationID.String(),
		ActorType:      "service",
		ActorLabel:     label,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID.String(),
		BeforeState:    before,
		AfterState:     after,
		Result:         "success",
		IP:             ip,
	})
}
