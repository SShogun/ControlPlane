-- Initial schema for ControlPlane
-- Core identity and notebook backbone

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE teams (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE memberships (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, team_id)
);

CREATE TABLE notebooks (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE notebook_revisions (
    id BIGSERIAL PRIMARY KEY,
    notebook_id BIGINT NOT NULL REFERENCES notebooks(id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE notebooks
    ADD COLUMN team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL;

ALTER TABLE notebooks
    ADD COLUMN slug TEXT UNIQUE;

ALTER TABLE notebooks
    ADD COLUMN visibility TEXT NOT NULL DEFAULT 'private';

ALTER TABLE notebooks
    ADD COLUMN current_published_revision_id BIGINT;

ALTER TABLE notebooks
    ADD CONSTRAINT notebooks_current_published_revision_fk
    FOREIGN KEY (current_published_revision_id)
    REFERENCES notebook_revisions(id)
    ON DELETE SET NULL;

CREATE TABLE tags (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE notebook_tags (
    notebook_id BIGINT NOT NULL REFERENCES notebooks(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (notebook_id, tag_id)
);

CREATE TABLE approvals (
    id BIGSERIAL PRIMARY KEY,
    notebook_revision_id BIGINT NOT NULL REFERENCES notebook_revisions(id) ON DELETE CASCADE,
    reviewer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    decision TEXT NOT NULL,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_events (
    id BIGSERIAL PRIMARY KEY,
    actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id BIGINT NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE moderation_flags (
    id BIGSERIAL PRIMARY KEY,
    notebook_id BIGINT NOT NULL REFERENCES notebooks(id) ON DELETE CASCADE,
    moderator_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT NOT NULL,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_memberships_team_id ON memberships(team_id);
CREATE INDEX idx_notebooks_team_id ON notebooks(team_id);
CREATE INDEX idx_notebooks_visibility ON notebooks(visibility);
CREATE INDEX idx_revisions_notebook_id ON notebook_revisions(notebook_id);
CREATE INDEX idx_revisions_status ON notebook_revisions(status);
CREATE INDEX idx_notebook_tags_tag_id ON notebook_tags(tag_id);
CREATE INDEX idx_approvals_revision_id ON approvals(notebook_revision_id);
CREATE INDEX idx_audit_events_entity ON audit_events(entity_type, entity_id);
CREATE INDEX idx_moderation_flags_notebook_id ON moderation_flags(notebook_id);
