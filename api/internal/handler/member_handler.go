package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/dokkiitech/ashiato/api/internal/domain"
	appctx "github.com/dokkiitech/ashiato/api/internal/middleware"
	"github.com/dokkiitech/ashiato/api/internal/usecase"
)

type createMemberRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type memberResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func RegisterMemberRoutes(g *echo.Group, service *usecase.Service) {
	g.POST("/members", createMemberHandler(service))
}

func createMemberHandler(service *usecase.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		actor, ok := appctx.ActorFromContext(c.Request().Context())
		if !ok {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"code":    "unauthorized",
				"message": "missing actor in context",
			})
		}

		var req createMemberRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"code":    "bad_request",
				"message": "invalid request body",
			})
		}

		member, err := service.CreateMember(c.Request().Context(), actor, usecase.CreateMemberInput{
			Email:       req.Email,
			Password:    req.Password,
			DisplayName: req.DisplayName,
		})
		if err != nil {
			return memberErrorResponse(c, err)
		}

		return c.JSON(http.StatusCreated, memberResponse{
			ID:    member.ID.String(),
			Email: member.Email,
			Name:  member.Name,
			Role:  member.Role,
		})
	}
}

func memberErrorResponse(c echo.Context, err error) error {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case domain.ErrorCodeForbidden:
			return c.JSON(http.StatusForbidden, map[string]string{
				"code":    string(appErr.Code),
				"message": appErr.Message,
			})
		case domain.ErrorCodeValidation:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"code":    string(appErr.Code),
				"message": appErr.Message,
			})
		}
	}
	return c.JSON(http.StatusInternalServerError, map[string]string{
		"code":    "internal",
		"message": err.Error(),
	})
}
