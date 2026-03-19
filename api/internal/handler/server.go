package handler

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/dokkiitech/ashiato/api/internal/domain"
	appctx "github.com/dokkiitech/ashiato/api/internal/middleware"
	"github.com/dokkiitech/ashiato/api/internal/oapi"
	"github.com/dokkiitech/ashiato/api/internal/usecase"
)

type Server struct {
	service *usecase.Service
}

func NewServer(service *usecase.Service) *Server {
	return &Server{service: service}
}

func (s *Server) GetMe(ctx context.Context, _ oapi.GetMeRequestObject) (oapi.GetMeResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return oapi.GetMe200JSONResponse{}, errors.New("missing actor in context")
	}

	currentActor, organization, err := s.service.GetMe(ctx, actor)
	if err != nil {
		return nil, err
	}

	return oapi.GetMe200JSONResponse{
		User: oapi.ActorUser{
			Id:      openapi_types.UUID(currentActor.UserID),
			Subject: currentActor.Subject,
			Email:   openapi_types.Email(currentActor.Email),
			Name:    currentActor.Name,
			Role:    oapi.OrganizationRole(currentActor.Role),
		},
		Organization: oapi.Organization{
			Id:   openapi_types.UUID(organization.ID),
			Slug: organization.Slug,
			Name: organization.Name,
		},
	}, nil
}

func (s *Server) GetMembers(ctx context.Context, _ oapi.GetMembersRequestObject) (oapi.GetMembersResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	members, err := s.service.ListMembers(ctx, actor)
	if err != nil {
		return nil, err
	}

	response := oapi.MembersResponse{Members: make([]oapi.Member, 0, len(members))}
	for _, member := range members {
		response.Members = append(response.Members, oapi.Member{
			Id:    openapi_types.UUID(member.ID),
			Email: openapi_types.Email(member.Email),
			Name:  member.Name,
			Role:  oapi.OrganizationRole(member.Role),
		})
	}
	return oapi.GetMembers200JSONResponse(response), nil
}

func (s *Server) GetWorkflowPeriods(ctx context.Context, _ oapi.GetWorkflowPeriodsRequestObject) (oapi.GetWorkflowPeriodsResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	periods, err := s.service.ListWorkflowPeriods(ctx, actor)
	if err != nil {
		return nil, err
	}

	response := oapi.WorkflowPeriodsResponse{Periods: make([]oapi.WorkflowPeriod, 0, len(periods))}
	for _, period := range periods {
		response.Periods = append(response.Periods, oapi.WorkflowPeriod{
			Year:                 int(period.Year),
			Month:                int(period.Month),
			MeetingStatus:        period.MeetingStatus,
			TaskCompletionStatus: period.TaskCompletionStatus,
			AnnouncementStatus:   period.AnnouncementStatus,
			NextStageReady:       period.NextStageReady,
		})
	}
	return oapi.GetWorkflowPeriods200JSONResponse(response), nil
}

func (s *Server) GetMeeting(ctx context.Context, request oapi.GetMeetingRequestObject) (oapi.GetMeetingResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	meeting, err := s.service.GetMeeting(ctx, actor, int32(request.Year), int32(request.Month))
	if err != nil {
		return responseForMeetingError(err), nil
	}
	return oapi.GetMeeting200JSONResponse(openapiMeeting(meeting)), nil
}

func (s *Server) PutMeeting(ctx context.Context, request oapi.PutMeetingRequestObject) (oapi.PutMeetingResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	meeting, err := s.service.PutMeeting(ctx, actor, usecase.PutMeetingInput{
		Year:        int32(request.Year),
		Month:       int32(request.Month),
		ScheduledAt: request.Body.ScheduledAt,
		MeetingURL:  request.Body.MeetingUrl,
		Notes:       request.Body.Notes,
		Status:      string(request.Body.Status),
	})
	if err != nil {
		return responseForPutMeetingError(err), nil
	}
	return oapi.PutMeeting200JSONResponse(openapiMeeting(meeting)), nil
}

func (s *Server) GetTasks(ctx context.Context, request oapi.GetTasksRequestObject) (oapi.GetTasksResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	tasks, err := s.service.ListTasks(ctx, actor, int32(request.Params.Year), int32(request.Params.Month))
	if err != nil {
		return responseForGetTasksError(err), nil
	}

	response := oapi.TasksResponse{Tasks: make([]oapi.Task, 0, len(tasks))}
	for _, task := range tasks {
		response.Tasks = append(response.Tasks, openapiTask(task))
	}
	return oapi.GetTasks200JSONResponse(response), nil
}

func (s *Server) PostTask(ctx context.Context, request oapi.PostTaskRequestObject) (oapi.PostTaskResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	task, err := s.service.CreateTask(ctx, actor, usecase.CreateTaskInput{
		Title:        request.Body.Title,
		Status:       string(request.Body.Status),
		DueDate:      dateFromOpenAPI(request.Body.DueDate),
		ReferenceURL: request.Body.ReferenceUrl,
		AssigneeID:   uuidPtrFromOpenAPI(request.Body.AssigneeId),
		ApproverIDs:  uuidsFromOpenAPI(request.Body.ApproverIds),
		Year:         int32(request.Body.Year),
		Month:        int32(request.Body.Month),
	})
	if err != nil {
		return responseForPostTaskError(err), nil
	}
	return oapi.PostTask201JSONResponse(openapiTask(task)), nil
}

func (s *Server) PatchTask(ctx context.Context, request oapi.PatchTaskRequestObject) (oapi.PatchTaskResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	task, err := s.service.UpdateTask(ctx, actor, usecase.UpdateTaskInput{
		ID:           uuid.UUID(request.Id),
		Title:        request.Body.Title,
		Status:       string(request.Body.Status),
		DueDate:      dateFromOpenAPI(request.Body.DueDate),
		ReferenceURL: request.Body.ReferenceUrl,
		AssigneeID:   uuidPtrFromOpenAPI(request.Body.AssigneeId),
		ApproverIDs:  uuidsFromOpenAPI(request.Body.ApproverIds),
		Version:      int32(request.Body.Version),
	})
	if err != nil {
		return responseForPatchTaskError(err), nil
	}
	return oapi.PatchTask200JSONResponse(openapiTask(task)), nil
}

func (s *Server) DeleteTask(ctx context.Context, request oapi.DeleteTaskRequestObject) (oapi.DeleteTaskResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	if err := s.service.DeleteTask(ctx, actor, uuid.UUID(request.Id), int32(request.Params.Version)); err != nil {
		return responseForDeleteTaskError(err), nil
	}
	return oapi.DeleteTask204Response{}, nil
}

func (s *Server) ApproveTask(ctx context.Context, request oapi.ApproveTaskRequestObject) (oapi.ApproveTaskResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	task, err := s.service.ApproveTask(ctx, actor, uuid.UUID(request.Id))
	if err != nil {
		return responseForApproveTaskError(err), nil
	}
	return oapi.ApproveTask200JSONResponse(openapiTask(task)), nil
}

func (s *Server) GetAnnouncement(ctx context.Context, request oapi.GetAnnouncementRequestObject) (oapi.GetAnnouncementResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	announcement, err := s.service.GetAnnouncement(ctx, actor, int32(request.Year), int32(request.Month))
	if err != nil {
		return responseForAnnouncementReadError(err), nil
	}
	return oapi.GetAnnouncement200JSONResponse(openapiAnnouncement(announcement)), nil
}

func (s *Server) PutAnnouncement(ctx context.Context, request oapi.PutAnnouncementRequestObject) (oapi.PutAnnouncementResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	announcement, err := s.service.PutAnnouncement(ctx, actor, usecase.PutAnnouncementInput{
		Year:           int32(request.Year),
		Month:          int32(request.Month),
		Body:           request.Body.Body,
		PublishChannel: request.Body.PublishChannel,
	})
	if err != nil {
		return responseForPutAnnouncementError(err), nil
	}
	return oapi.PutAnnouncement200JSONResponse(openapiAnnouncement(announcement)), nil
}

func (s *Server) PublishAnnouncement(ctx context.Context, request oapi.PublishAnnouncementRequestObject) (oapi.PublishAnnouncementResponseObject, error) {
	actor, ok := appctx.ActorFromContext(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}

	announcement, err := s.service.PublishAnnouncement(ctx, actor, usecase.PublishAnnouncementInput{
		ID:             uuid.UUID(request.Id),
		PublishChannel: request.Body.PublishChannel,
	})
	if err != nil {
		return responseForPublishAnnouncementError(err), nil
	}
	return oapi.PublishAnnouncement200JSONResponse(openapiAnnouncement(announcement)), nil
}

func (s *Server) GetAnnouncementPublishRequests(ctx context.Context, _ oapi.GetAnnouncementPublishRequestsRequestObject) (oapi.GetAnnouncementPublishRequestsResponseObject, error) {
	requests, err := s.service.ListPublishRequests(ctx)
	if err != nil {
		return nil, err
	}

	response := oapi.PublishRequestsResponse{Requests: make([]oapi.PublishRequest, 0, len(requests))}
	for _, request := range requests {
		response.Requests = append(response.Requests, oapi.PublishRequest{
			AnnouncementId: openapi_types.UUID(request.AnnouncementID),
			Year:           int(request.Year),
			Month:          int(request.Month),
			Body:           request.Body,
			PublishChannel: request.PublishChannel,
		})
	}
	return oapi.GetAnnouncementPublishRequests200JSONResponse(response), nil
}

func (s *Server) CompleteAnnouncementPublishRequest(ctx context.Context, request oapi.CompleteAnnouncementPublishRequestRequestObject) (oapi.CompleteAnnouncementPublishRequestResponseObject, error) {
	announcement, err := s.service.CompletePublishRequest(ctx, uuid.UUID(request.Id), request.Body.DiscordMessageId)
	if err != nil {
		return responseForCompleteAnnouncementPublishError(err), nil
	}
	return oapi.CompleteAnnouncementPublishRequest200JSONResponse(openapiAnnouncement(announcement)), nil
}

func (s *Server) FailAnnouncementPublishRequest(ctx context.Context, request oapi.FailAnnouncementPublishRequestRequestObject) (oapi.FailAnnouncementPublishRequestResponseObject, error) {
	announcement, err := s.service.FailPublishRequest(ctx, uuid.UUID(request.Id), request.Body.Error)
	if err != nil {
		return responseForFailAnnouncementPublishError(err), nil
	}
	return oapi.FailAnnouncementPublishRequest200JSONResponse(openapiAnnouncement(announcement)), nil
}

func dateFromOpenAPI(value *openapi_types.Date) *time.Time {
	if value == nil {
		return nil
	}
	parsed := value.Time
	return &parsed
}

func uuidPtrFromOpenAPI(value *openapi_types.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	parsed := uuid.UUID(*value)
	return &parsed
}

func uuidsFromOpenAPI(values []openapi_types.UUID) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		result = append(result, uuid.UUID(value))
	}
	return result
}

func responseForMeetingError(err error) oapi.GetMeetingResponseObject {
	var appErr *domain.AppError
	if errors.As(err, &appErr) && appErr.Code == domain.ErrorCodeNotFound {
		return oapi.GetMeeting404JSONResponse{NotFoundJSONResponse: oapi.NotFoundJSONResponse(errorResponse(err))}
	}
	return oapi.GetMeeting404JSONResponse{NotFoundJSONResponse: oapi.NotFoundJSONResponse(errorResponse(err))}
}

func responseForPutMeetingError(err error) oapi.PutMeetingResponseObject {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case domain.ErrorCodeForbidden:
			return oapi.PutMeeting403JSONResponse{ForbiddenJSONResponse: oapi.ForbiddenJSONResponse(errorResponse(err))}
		case domain.ErrorCodeValidation:
			return oapi.PutMeeting400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
		}
	}
	return oapi.PutMeeting400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
}

func responseForGetTasksError(err error) oapi.GetTasksResponseObject {
	return oapi.GetTasks400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
}

func responseForPostTaskError(err error) oapi.PostTaskResponseObject {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case domain.ErrorCodeForbidden:
			return oapi.PostTask403JSONResponse{ForbiddenJSONResponse: oapi.ForbiddenJSONResponse(errorResponse(err))}
		case domain.ErrorCodeValidation:
			return oapi.PostTask400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
		}
	}
	return oapi.PostTask400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
}

func responseForPatchTaskError(err error) oapi.PatchTaskResponseObject {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case domain.ErrorCodeForbidden:
			return oapi.PatchTask403JSONResponse{ForbiddenJSONResponse: oapi.ForbiddenJSONResponse(errorResponse(err))}
		case domain.ErrorCodeValidation:
			return oapi.PatchTask400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
		case domain.ErrorCodeNotFound:
			return oapi.PatchTask404JSONResponse{NotFoundJSONResponse: oapi.NotFoundJSONResponse(errorResponse(err))}
		case domain.ErrorCodeConflict:
			return oapi.PatchTask409JSONResponse{ConflictJSONResponse: oapi.ConflictJSONResponse(errorResponse(err))}
		}
	}
	return oapi.PatchTask400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
}

func responseForDeleteTaskError(err error) oapi.DeleteTaskResponseObject {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case domain.ErrorCodeForbidden:
			return oapi.DeleteTask403JSONResponse{ForbiddenJSONResponse: oapi.ForbiddenJSONResponse(errorResponse(err))}
		case domain.ErrorCodeValidation:
			return oapi.DeleteTask400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
		case domain.ErrorCodeNotFound:
			return oapi.DeleteTask404JSONResponse{NotFoundJSONResponse: oapi.NotFoundJSONResponse(errorResponse(err))}
		case domain.ErrorCodeConflict:
			return oapi.DeleteTask409JSONResponse{ConflictJSONResponse: oapi.ConflictJSONResponse(errorResponse(err))}
		}
	}
	return oapi.DeleteTask400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
}

func responseForApproveTaskError(err error) oapi.ApproveTaskResponseObject {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case domain.ErrorCodeForbidden:
			return oapi.ApproveTask403JSONResponse{ForbiddenJSONResponse: oapi.ForbiddenJSONResponse(errorResponse(err))}
		case domain.ErrorCodeNotFound:
			return oapi.ApproveTask404JSONResponse{NotFoundJSONResponse: oapi.NotFoundJSONResponse(errorResponse(err))}
		case domain.ErrorCodeConflict:
			return oapi.ApproveTask409JSONResponse{ConflictJSONResponse: oapi.ConflictJSONResponse(errorResponse(err))}
		}
	}
	return oapi.ApproveTask409JSONResponse{ConflictJSONResponse: oapi.ConflictJSONResponse(errorResponse(err))}
}

func responseForAnnouncementReadError(err error) oapi.GetAnnouncementResponseObject {
	return oapi.GetAnnouncement404JSONResponse{NotFoundJSONResponse: oapi.NotFoundJSONResponse(errorResponse(err))}
}

func responseForPutAnnouncementError(err error) oapi.PutAnnouncementResponseObject {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case domain.ErrorCodeForbidden:
			return oapi.PutAnnouncement403JSONResponse{ForbiddenJSONResponse: oapi.ForbiddenJSONResponse(errorResponse(err))}
		case domain.ErrorCodeValidation:
			return oapi.PutAnnouncement400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
		}
	}
	return oapi.PutAnnouncement400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
}

func responseForPublishAnnouncementError(err error) oapi.PublishAnnouncementResponseObject {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case domain.ErrorCodeForbidden:
			return oapi.PublishAnnouncement403JSONResponse{ForbiddenJSONResponse: oapi.ForbiddenJSONResponse(errorResponse(err))}
		case domain.ErrorCodeNotFound:
			return oapi.PublishAnnouncement404JSONResponse{NotFoundJSONResponse: oapi.NotFoundJSONResponse(errorResponse(err))}
		case domain.ErrorCodeValidation:
			return oapi.PublishAnnouncement400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
		case domain.ErrorCodeConflict:
			return oapi.PublishAnnouncement400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
		}
	}
	return oapi.PublishAnnouncement400JSONResponse{BadRequestJSONResponse: oapi.BadRequestJSONResponse(errorResponse(err))}
}

func responseForCompleteAnnouncementPublishError(err error) oapi.CompleteAnnouncementPublishRequestResponseObject {
	return oapi.CompleteAnnouncementPublishRequest404JSONResponse{NotFoundJSONResponse: oapi.NotFoundJSONResponse(errorResponse(err))}
}

func responseForFailAnnouncementPublishError(err error) oapi.FailAnnouncementPublishRequestResponseObject {
	return oapi.FailAnnouncementPublishRequest404JSONResponse{NotFoundJSONResponse: oapi.NotFoundJSONResponse(errorResponse(err))}
}
