package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/emzola/issuetracker/internal/repository"
	"github.com/emzola/issuetracker/pkg/model"
)

func (r *Repository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (name, email, password_hash, activated, role, created_by, modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_on, modified_on, version`
	args := []interface{}{user.Name, user.Email, user.Password.Hash, user.Activated, user.Role, user.CreatedBy, user.ModifiedBy}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedOn, &user.ModifiedOn, &user.Version)
	if err != nil {
		switch {
		case err.Error() == "ERROR: canceling statement due to user request":
			return fmt.Errorf("%v: %w", err, ctx.Err())
		case err.Error() == `ERROR: duplicate key value violates unique constraint "users_email_key" (SQLSTATE 23505)`:
			return repository.ErrDuplicateKey
		default:
			return err
		}
	}
	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, name, email, password_hash, activated, role, created_on, created_by, modified_on, modified_by, version
		FROM users
		WHERE email = $1`
	var user model.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.Role,
		&user.CreatedOn,
		&user.CreatedBy,
		&user.ModifiedOn,
		&user.ModifiedBy,
		&user.Version,
	)
	if err != nil {
		switch {
		case err.Error() == "ERROR: canceling statement due to user request":
			return nil, fmt.Errorf("%v: %w", err, ctx.Err())
		case errors.Is(err, sql.ErrNoRows):
			return nil, repository.ErrNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	query := `
		SELECT id, name, email, password_hash, activated, role, created_on, created_by, modified_on, modified_by, version
		FROM users
		WHERE id = $1`
	var user model.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.Role,
		&user.CreatedOn,
		&user.CreatedBy,
		&user.ModifiedOn,
		&user.ModifiedBy,
		&user.Version,
	)
	if err != nil {
		switch {
		case err.Error() == "ERROR: canceling statement due to user request":
			return nil, fmt.Errorf("%v: %w", err, ctx.Err())
		case errors.Is(err, sql.ErrNoRows):
			return nil, repository.ErrNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (r *Repository) GetAllUsers(ctx context.Context, name, email, role string, filters model.Filters) ([]*model.User, model.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, name, email, password_hash, activated, role, created_on, created_by, modified_on, modified_by, version
		FROM users
		WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
		AND (LOWER(email) = LOWER($2) OR $2 = '')
		AND (LOWER(role) = LOWER($3) OR $3 = '')
		ORDER BY %s %s, id ASC 
		LIMIT $4 OFFSET $5`, filters.SortColumn(), filters.SortDirection())
	args := []interface{}{name, email, role, filters.Limit(), filters.Offset()}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == "ERROR: canceling statement due to user request":
			return nil, model.Metadata{}, fmt.Errorf("%v: %w", err, ctx.Err())
		default:
			return nil, model.Metadata{}, err
		}
	}
	defer rows.Close()
	totalRecords := 0
	users := []*model.User{}
	for rows.Next() {
		var user model.User
		err := rows.Scan(
			&totalRecords,
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Password.Hash,
			&user.Activated,
			&user.Role,
			&user.CreatedOn,
			&user.CreatedBy,
			&user.ModifiedOn,
			&user.ModifiedBy,
			&user.Version,
		)
		if err != nil {
			return nil, model.Metadata{}, err
		}
		users = append(users, &user)
	}
	if err = rows.Err(); err != nil {
		return nil, model.Metadata{}, err
	}
	metadata := model.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return users, metadata, nil
}

func (r *Repository) UpdateUser(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, activated = $4, role = $5, version = version + 1
		WHERE id = $6 AND version = $7
		RETURNING version`
	args := []interface{}{user.Name, user.Email, user.Password.Hash, user.Activated, user.Role, user.ID, user.Version}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case err.Error() == "ERROR: canceling statement due to user request":
			return fmt.Errorf("%v: %w", err, ctx.Err())
		case err.Error() == `ERROR: duplicate key value violates unique constraint "users_email_key" (SQLSTATE 23505)`:
			return repository.ErrDuplicateKey
		case errors.Is(err, sql.ErrNoRows):
			return repository.ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

func (r *Repository) GetUserForToken(ctx context.Context, tokenScope, tokenPlaintext string) (*model.User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	query := `
		SELECT users.id, users.name, users.email, users.password_hash, users.activated, users.role, users.created_on, users.created_by, users.modified_on, users.modified_by, users.version
		FROM users
		INNER JOIN tokens
		ON users.id = tokens.user_id
		WHERE tokens.hash = $1
		AND tokens.scope = $2 
		AND tokens.expiry > $3`
	args := []interface{}{tokenHash[:], tokenScope, time.Now()}
	var user model.User
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.Role,
		&user.CreatedOn,
		&user.CreatedBy,
		&user.ModifiedOn,
		&user.ModifiedBy,
		&user.Version,
	)
	if err != nil {
		switch {
		case err.Error() == "ERROR: canceling statement due to user request":
			return nil, fmt.Errorf("%v: %w", err, ctx.Err())
		case errors.Is(err, sql.ErrNoRows):
			return nil, repository.ErrNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (r *Repository) DeleteUser(ctx context.Context, id int64) error {
	if id < 1 {
		return repository.ErrNotFound
	}
	query := `
		DELETE FROM users
		WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		switch {
		case err.Error() == "ERROR: canceling statement due to user request":
			return fmt.Errorf("%v: %w", err, ctx.Err())
		default:
			return err
		}
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *Repository) AssignUserToProject(ctx context.Context, userID, projectID int64) error {
	query := `
		INSERT INTO projects_users 
		SELECT $1, users.id FROM users WHERE users.id = $2`
	args := []interface{}{projectID, userID}
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == "ERROR: canceling statement due to user request":
			return fmt.Errorf("%v: %w", err, ctx.Err())
		case err.Error() == `ERROR: duplicate key value violates unique constraint "projects_users_pkey" (SQLSTATE 23505)`:
			return repository.ErrDuplicateKey
		default:
			return err
		}

	}
	return nil
}
