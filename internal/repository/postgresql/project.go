package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emzola/bugtracker/internal/model"
	"github.com/emzola/bugtracker/internal/repository"
)

// CreateProject adds a new project record.
func (r *Repository) CreateProject(ctx context.Context, project *model.Project) error {
	var query string
	var args []interface{}
	switch {
	case project.StartDate == nil && project.EndDate == nil:
		query = `
			INSERT INTO project (name, description, owner, status, public_access, created_by, modified_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, created_on, last_modified, version`
		args = []interface{}{project.Name, project.Description, project.Owner, project.Status, project.PublicAccess, project.CreatedBy, project.ModifiedBy}
	case project.EndDate == nil:
		query = `
			INSERT INTO project (name, description, owner, status, start_date, public_access, created_by, modified_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, created_on, last_modified, version`
		args = []interface{}{project.Name, project.Description, project.Owner, project.Status, *project.StartDate, project.PublicAccess, project.CreatedBy, project.ModifiedBy}
	default:
		query = `
			INSERT INTO project (name, description, owner, status, start_date, end_date, public_access, created_by, modified_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id, created_on, last_modified, version`
		args = []interface{}{project.Name, project.Description, project.Owner, project.Status, *project.StartDate, *project.EndDate, project.PublicAccess, project.CreatedBy, project.ModifiedBy}
	}
	return r.db.QueryRowContext(ctx, query, args...).Scan(&project.ID, &project.CreatedOn, &project.LastModified, &project.Version)
}

// GetProject retrieves a project record by its id.
func (r *Repository) GetProject(ctx context.Context, id int64) (*model.Project, error) {
	if id < 1 {
		return nil, repository.ErrNotFound
	}
	query := `
		SELECT id, name, description, owner, status, start_date, end_date, completed_on, created_on, last_modified, created_by, modified_by, public_access, version
		FROM project
		WHERE id = $1`
	var project model.Project
	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.Owner,
		&project.Status,
		&project.StartDate,
		&project.EndDate,
		&project.CompletedOn,
		&project.CreatedOn,
		&project.LastModified,
		&project.CreatedBy,
		&project.ModifiedBy,
		&project.PublicAccess,
		&project.Version,
	); err != nil {
		switch {
		case err.Error() == "pq: canceling statement due to user request":
			return nil, fmt.Errorf("%v: %w", err, ctx.Err())
		case errors.Is(err, sql.ErrNoRows):
			return nil, repository.ErrNotFound
		default:
			return nil, err
		}
	}
	return &project, nil
}
