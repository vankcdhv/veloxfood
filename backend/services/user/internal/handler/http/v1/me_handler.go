package v1

import (
	"time"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// MeHandler handles self-service profile endpoints.
type MeHandler struct {
	profileUC usecase.ProfileUsecase
}

func NewMeHandler(profileUC usecase.ProfileUsecase) *MeHandler {
	return &MeHandler{profileUC: profileUC}
}

// GetMe returns GET /api/v1/me — full profile bundle.
func (h *MeHandler) GetMe(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())
	bundle, err := h.profileUC.GetMe(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, toProfileBundleResponse(bundle))
}

// updateMeRequest — whitelisted self-service user fields.
type updateMeRequest struct {
	FullName  *string        `json:"full_name"`
	DOB       *string        `json:"dob"` // ISO-8601 date string YYYY-MM-DD
	Gender    *entity.Gender `json:"gender"`
	AvatarURL *string        `json:"avatar_url"`
}

// UpdateMe handles PATCH /api/v1/me.
func (h *MeHandler) UpdateMe(c *gin.Context) {
	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	input := usecase.UpdateMeInput{
		FullName:  req.FullName,
		Gender:    req.Gender,
		AvatarURL: req.AvatarURL,
	}

	if req.DOB != nil {
		t, err := time.Parse("2006-01-02", *req.DOB)
		if err != nil {
			response.BadRequest(c, "dob must be YYYY-MM-DD format")
			return
		}
		input.DOB = &t
	}

	user, err := h.profileUC.UpdateMe(c.Request.Context(), authmw.UserIDFromContext(c.Request.Context()), input)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, toUserResponse(user))
}

// ---- response helpers ----

type profileBundleResponse struct {
	User              *userResponse              `json:"user"`
	Roles             []*userRoleResponse        `json:"roles"`
	VendorMemberships []*entity.VendorMembership `json:"vendor_memberships"`
}

type userRoleResponse struct {
	RoleID    string           `json:"role_id"`
	ScopeType entity.ScopeType `json:"scope_type"`
	ScopeID   *string          `json:"scope_id"`
	RoleCode  *string          `json:"role_code"`
}

func toProfileBundleResponse(b *usecase.ProfileBundle) profileBundleResponse {
	roles := make([]*userRoleResponse, len(b.Roles))
	for i, r := range b.Roles {
		rr := &userRoleResponse{
			RoleID:    r.RoleID,
			ScopeType: r.ScopeType,
			ScopeID:   r.ScopeID,
		}
		if r.Role != nil {
			rr.RoleCode = &r.Role.Code
		}
		roles[i] = rr
	}
	u := toUserResponse(b.User)
	return profileBundleResponse{
		User:              &u,
		Roles:             roles,
		VendorMemberships: b.VendorMemberships,
	}
}
