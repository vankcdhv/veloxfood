package usecase

import (
	"context"
	"log/slog"

	"project/pkg/audit"
	"project/services/user/internal/entity"
)

// UpdateStudentByAdmin allows admin to update student-specific profile fields.
// Uses Upsert so it works regardless of whether profile row exists yet.
func (uc *adminUserUsecase) UpdateStudentByAdmin(
	ctx context.Context,
	adminID, userID string,
	input UpdateStudentAdminInput,
) error {
	if _, err := uc.userRepo.GetByID(ctx, userID); err != nil {
		return ErrUserNotFound
	}

	profile := &entity.StudentProfile{
		UserID:        userID,
		StudentCode:   input.StudentCode,
		Faculty:       input.Faculty,
		Class:         input.Class,
		CohortYear:    input.CohortYear,
		DormitoryRoom: input.DormitoryRoom,
		Allergies:     input.Allergies,
	}

	if err := uc.studentRepo.Upsert(ctx, profile); err != nil {
		slog.ErrorContext(ctx, "admin update student profile failed",
			"admin_id", adminID, "user_id", userID, "err", err)
		return err
	}

	slog.InfoContext(ctx, "admin updated student profile", "admin_id", adminID, "user_id", userID)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &adminID,
		Action:      "admin.profile_updated",
		TargetType:  "user",
		TargetID:    &userID,
		Payload:     map[string]string{"profile_type": "student"},
	})
	return nil
}

// UpdateFacultyByAdmin allows admin to update faculty-specific profile fields.
func (uc *adminUserUsecase) UpdateFacultyByAdmin(
	ctx context.Context,
	adminID, userID string,
	input UpdateFacultyAdminInput,
) error {
	if _, err := uc.userRepo.GetByID(ctx, userID); err != nil {
		return ErrUserNotFound
	}

	profile := &entity.FacultyProfile{
		UserID:     userID,
		StaffCode:  input.StaffCode,
		Department: input.Department,
		Position:   input.Position,
	}
	if input.AllowPayrollDeduction != nil {
		profile.AllowPayrollDeduction = *input.AllowPayrollDeduction
	}

	if err := uc.facultyRepo.Upsert(ctx, profile); err != nil {
		slog.ErrorContext(ctx, "admin update faculty profile failed",
			"admin_id", adminID, "user_id", userID, "err", err)
		return err
	}

	slog.InfoContext(ctx, "admin updated faculty profile", "admin_id", adminID, "user_id", userID)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &adminID,
		Action:      "admin.profile_updated",
		TargetType:  "user",
		TargetID:    &userID,
		Payload:     map[string]string{"profile_type": "faculty"},
	})
	return nil
}
