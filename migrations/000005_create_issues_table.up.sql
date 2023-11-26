CREATE TABLE IF NOT EXISTS issues (
    issue_id bigserial PRIMARY KEY,
    title text NOT NULL,
    description text NOT NULL DEFAULT '',
    reporter_id bigint NOT NULL REFERENCES users,
    reported_date date NOT NULL,
    project_id bigint NOT NULL REFERENCES projects ON DELETE CASCADE,
    assigned_to bigint REFERENCES users,
    status text NOT NULL,
    priority text NOT NULL,
    target_resolution_date date NOT NULL,
    progress text NOT NULL DEFAULT '',
    actual_Resolution_date date,
    resolution_summary text NOT NULL DEFAULT '',
    created_on timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    created_by text NOT NULL,
    modified_on timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    modified_by text NOT NULL
);