package audit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dokkiitech/ashiato/api/internal/db"
	"github.com/dokkiitech/ashiato/api/internal/domain"
)

func WriteUser(
	ctx context.Context,
	q *db.Queries,
	actor domain.Actor,
	ip string,
	action string,
	resourceType string,
	resourceID uuid.UUID,
	before any,
	after any,
) error {
	return write(ctx, q, actor.OrganizationID, &actor.UserID, "user", actor.Email, ip, action, resourceType, resourceID, before, after)
}

func WriteService(
	ctx context.Context,
	q *db.Queries,
	organizationID uuid.UUID,
	label string,
	ip string,
	action string,
	resourceType string,
	resourceID uuid.UUID,
	before any,
	after any,
) error {
	return write(ctx, q, organizationID, nil, "service", label, ip, action, resourceType, resourceID, before, after)
}

func write(
	ctx context.Context,
	q *db.Queries,
	organizationID uuid.UUID,
	actorUserID *uuid.UUID,
	actorType string,
	actorLabel string,
	ip string,
	action string,
	resourceType string,
	resourceID uuid.UUID,
	before any,
	after any,
) error {
	beforeState, err := marshalOptional(before)
	if err != nil {
		return err
	}
	afterState, err := marshalOptional(after)
	if err != nil {
		return err
	}

	var actorUserIDValue pgtype.UUID
	if actorUserID != nil {
		actorUserIDValue = pgtype.UUID{Bytes: [16]byte(*actorUserID), Valid: true}
	}

	return q.CreateAuditLog(ctx, db.CreateAuditLogParams{
		OrganizationID: pgtype.UUID{Bytes: [16]byte(organizationID), Valid: true},
		ActorUserID:    actorUserIDValue,
		ActorType:      actorType,
		ActorLabel:     pgtype.Text{String: actorLabel, Valid: actorLabel != ""},
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     pgtype.UUID{Bytes: [16]byte(resourceID), Valid: true},
		BeforeState:    beforeState,
		AfterState:     afterState,
		Result:         "success",
		Ip:             pgtype.Text{String: ip, Valid: ip != ""},
	})
}

func marshalOptional(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}
