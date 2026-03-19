package handler

import (
	"errors"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/dokkiitech/ashiato/api/internal/domain"
	"github.com/dokkiitech/ashiato/api/internal/oapi"
)

func errorResponse(err error) oapi.ErrorResponse {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		return oapi.ErrorResponse{
			Code:    string(appErr.Code),
			Message: appErr.Message,
		}
	}
	return oapi.ErrorResponse{
		Code:    string(domain.ErrorCodeInternal),
		Message: "internal server error",
	}
}

func openapiTask(task domain.Task) oapi.Task {
	return oapi.Task{
		ApprovalState:       oapi.ApprovalState(task.ApprovalState),
		ApprovedApproverIds: openapiUUIDs(task.ApprovedApproverIDs),
		ApproverIds:         openapiUUIDs(task.ApproverIDs),
		AssigneeId:          openapiUUIDPtr(task.AssigneeID),
		CreatedBy:           openapiUUIDPtr(&task.CreatedBy),
		DueDate:             openapiDate(task.DueDate),
		Id:                  openapi_types.UUID(task.ID),
		Month:               int(task.Month),
		ReferenceUrl:        task.ReferenceURL,
		Status:              oapi.TaskStatus(task.Status),
		Title:               task.Title,
		Version:             int(task.Version),
		Year:                int(task.Year),
	}
}

func openapiMeeting(meeting domain.Meeting) oapi.Meeting {
	return oapi.Meeting{
		Id:          openapi_types.UUID(meeting.ID),
		MeetingUrl:  meeting.MeetingURL,
		Month:       int(meeting.Month),
		Notes:       meeting.Notes,
		ScheduledAt: meeting.ScheduledAt,
		Status:      oapi.MeetingStatus(meeting.Status),
		Year:        int(meeting.Year),
	}
}

func openapiAnnouncement(announcement domain.Announcement) oapi.Announcement {
	return oapi.Announcement{
		Body:             announcement.Body,
		DiscordMessageId: announcement.DiscordMessageID,
		Id:               openapi_types.UUID(announcement.ID),
		LastError:        announcement.LastError,
		Month:            int(announcement.Month),
		PublishChannel:   announcement.PublishChannel,
		Status:           oapi.AnnouncementStatus(announcement.Status),
		Year:             int(announcement.Year),
	}
}

func openapiUUIDs(values []uuid.UUID) []openapi_types.UUID {
	result := make([]openapi_types.UUID, 0, len(values))
	for _, value := range values {
		result = append(result, openapi_types.UUID(value))
	}
	return result
}

func openapiUUIDPtr(value *uuid.UUID) *openapi_types.UUID {
	if value == nil {
		return nil
	}
	result := openapi_types.UUID(*value)
	return &result
}

func openapiDate(value *time.Time) *openapi_types.Date {
	if value == nil {
		return nil
	}
	result := openapi_types.Date{Time: *value}
	return &result
}
