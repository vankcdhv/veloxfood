package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"project/services/user/internal/entity"
)

// ListUsers queries users with optional filters. Uses db.Raw for LEFT JOIN clarity.
// Default page_size=20, max=100. Total count is a separate query.
func (uc *adminUserUsecase) ListUsers(ctx context.Context, filter ListUsersFilter) ([]*entity.User, int64, error) {
	page := filter.Page
	pageSize := filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Build WHERE conditions dynamically for raw query.
	baseQuery := buildListUsersWhere(filter)

	// Count query.
	countSQL := `
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN vendor_memberships vm ON vm.user_id = u.id
		` + baseQuery.where
	var total int64
	if err := uc.db.WithContext(ctx).
		Raw(countSQL, baseQuery.args...).
		Scan(&total).Error; err != nil {
		slog.ErrorContext(ctx, "list users: count query failed", "err", err)
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	if total == 0 {
		return []*entity.User{}, 0, nil
	}

	// Data query.
	dataSQL := `
		SELECT DISTINCT u.id, u.email, u.phone, u.password_hash, u.status, u.full_name,
		       u.dob, u.gender, u.avatar_url, u.failed_login_attempts, u.locked_until,
		       u.created_at, u.updated_at
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN vendor_memberships vm ON vm.user_id = u.id
		` + baseQuery.where + `
		ORDER BY u.created_at DESC
		LIMIT ? OFFSET ?`

	args := append(baseQuery.args, pageSize, offset)
	var users []*entity.User
	if err := uc.db.WithContext(ctx).Raw(dataSQL, args...).Scan(&users).Error; err != nil {
		slog.ErrorContext(ctx, "list users: data query failed", "err", err)
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	return users, total, nil
}

type listWhereClause struct {
	where string
	args  []any
}

func buildListUsersWhere(f ListUsersFilter) listWhereClause {
	where := "WHERE 1=1"
	args := []any{}

	if f.RoleID != "" {
		where += " AND ur.role_id = ?"
		args = append(args, f.RoleID)
	}
	if f.VendorID != "" {
		where += " AND vm.vendor_id = ?"
		args = append(args, f.VendorID)
	}
	if f.Status != "" {
		where += " AND u.status = ?"
		args = append(args, f.Status)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		where += " AND (u.email ILIKE ? OR u.phone ILIKE ? OR u.full_name ILIKE ?)"
		args = append(args, like, like, like)
	}

	return listWhereClause{where: where, args: args}
}
