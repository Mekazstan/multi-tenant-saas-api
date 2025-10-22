-- +goose Up
-- +goose StatementBegin

-- Create token type enum
CREATE TYPE token_type AS ENUM ('email_verification', 'password_reset', 'team_invitation');

-- Add email_verified field to users table
ALTER TABLE users ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMP;

-- Create auth_tokens table for verification and reset tokens
CREATE TABLE auth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    type token_type NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for auth_tokens
CREATE INDEX idx_auth_tokens_token ON auth_tokens(token);
CREATE INDEX idx_auth_tokens_user_id ON auth_tokens(user_id);
CREATE INDEX idx_auth_tokens_type ON auth_tokens(type);
CREATE INDEX idx_auth_tokens_expires_at ON auth_tokens(expires_at);

-- Create team_invitations table
CREATE TABLE team_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role user_role NOT NULL,
    invited_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    accepted_at TIMESTAMP,
    declined_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for team_invitations
CREATE INDEX idx_team_invitations_token ON team_invitations(token);
CREATE INDEX idx_team_invitations_email ON team_invitations(email);
CREATE INDEX idx_team_invitations_organization_id ON team_invitations(organization_id);
CREATE INDEX idx_team_invitations_expires_at ON team_invitations(expires_at);

-- Add unique constraint to prevent duplicate pending invitations
CREATE UNIQUE INDEX idx_unique_pending_invitation ON team_invitations(organization_id, email) 
WHERE accepted_at IS NULL AND declined_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_unique_pending_invitation;
DROP TABLE IF EXISTS team_invitations;
DROP TABLE IF EXISTS auth_tokens;
DROP TYPE IF EXISTS token_type;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;

-- +goose StatementEnd