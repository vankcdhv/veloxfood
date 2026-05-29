package persistence

import (
	"context"
	"errors"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

type userGormRepository struct {
	db *gorm.DB
}

func NewUserGormRepository(db *gorm.DB) repository.UserRepository {
	return &userGormRepository{db: db}
}

func (r *userGormRepository) Create(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userGormRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userGormRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userGormRepository) Update(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userGormRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entity.User{}).Error
}

func (r *userGormRepository) List(ctx context.Context, offset, limit int) ([]*entity.User, int64, error) {
	var users []*entity.User
	var total int64

	if err := r.db.WithContext(ctx).Model(&entity.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// GetByIDs returns users matching the given UUIDs using ANY($1) for efficient batch lookup.
func (r *userGormRepository) GetByIDs(ctx context.Context, ids []string) ([]*entity.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []*entity.User
	if err := r.db.WithContext(ctx).Where("id = ANY(?)", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// FindByEmailOrPhone checks email first, then phone.
func (r *userGormRepository) FindByEmailOrPhone(ctx context.Context, identifier string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Where("email = ? OR phone = ?", identifier, identifier).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userGormRepository) UpdateStatus(ctx context.Context, id string, status entity.UserStatus) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// IncrementFailedAttempts uses atomic SQL to avoid lost-update race.
func (r *userGormRepository) IncrementFailedAttempts(ctx context.Context, id string) (int, error) {
	var count int
	res := r.db.WithContext(ctx).Raw(
		`UPDATE users SET failed_login_attempts = failed_login_attempts + 1 WHERE id = ? RETURNING failed_login_attempts`,
		id,
	).Scan(&count)
	if res.Error != nil {
		return 0, res.Error
	}
	return count, nil
}

func (r *userGormRepository) ResetFailedAttempts(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"failed_login_attempts": 0,
			"locked_until":          nil,
		}).Error
}

func (r *userGormRepository) SetLockedUntil(ctx context.Context, id string, until time.Time) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"locked_until":          until,
			"failed_login_attempts": 0,
		}).Error
}

func (r *userGormRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Update("password_hash", passwordHash).Error
}
