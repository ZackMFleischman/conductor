ALTER TABLE team_launches ADD COLUMN challenge_generation INTEGER NOT NULL DEFAULT 0 CHECK(challenge_generation >= 0);
ALTER TABLE team_launches ADD COLUMN acknowledged_generation INTEGER NOT NULL DEFAULT 0 CHECK(acknowledged_generation >= 0);
