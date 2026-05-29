package v1

import (
	"time"

	"project/services/user/internal/entity"
)

// userResponse is the standard HTTP response shape for a user record.
type userResponse struct {
	ID        string            `json:"id"`
	Email     *string           `json:"email"`
	FullName  string            `json:"full_name"`
	Phone     *string           `json:"phone"`
	Status    entity.UserStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func toUserResponse(u *entity.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		Phone:     u.Phone,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func toUserListResponse(users []*entity.User) []userResponse {
	result := make([]userResponse, len(users))
	for i, u := range users {
		result[i] = toUserResponse(u)
	}
	return result
}
