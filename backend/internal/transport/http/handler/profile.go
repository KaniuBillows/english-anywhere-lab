package handler

import (
	"net/http"

	"github.com/bennyshi/english-anywhere-lab/internal/auth"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/dto"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/middleware"
)

type ProfileHandler struct {
	svc *auth.Service
}

func NewProfileHandler(svc *auth.Service) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

func (h *ProfileHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	user, profile, err := h.svc.GetMe(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to get user")
		return
	}

	resp := dto.MeResponse{
		User: dto.UserDTO{
			ID:       user.ID,
			Email:    user.Email,
			Locale:   user.Locale,
			Timezone: user.Timezone,
		},
	}
	if profile != nil {
		resp.LearningProfile = dto.LearningProfileDTO{
			CurrentLevel:   profile.CurrentLevel,
			TargetDomain:   profile.TargetDomain,
			DailyMinutes:   profile.DailyMinutes,
			WeeklyGoalDays: profile.WeeklyGoalDays,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req dto.UpdateProfileRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	user, profile, err := h.svc.UpdateProfile(r.Context(), userID, req.CurrentLevel, req.TargetDomain, req.DailyMinutes, req.WeeklyGoalDays)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to update profile")
		return
	}

	resp := dto.MeResponse{
		User: dto.UserDTO{
			ID:       user.ID,
			Email:    user.Email,
			Locale:   user.Locale,
			Timezone: user.Timezone,
		},
	}
	if profile != nil {
		resp.LearningProfile = dto.LearningProfileDTO{
			CurrentLevel:   profile.CurrentLevel,
			TargetDomain:   profile.TargetDomain,
			DailyMinutes:   profile.DailyMinutes,
			WeeklyGoalDays: profile.WeeklyGoalDays,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
