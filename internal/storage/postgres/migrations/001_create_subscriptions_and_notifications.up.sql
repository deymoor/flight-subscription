CREATE TABLE subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    direction_from TEXT NOT NULL,
    direction_to TEXT NOT NULL,
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    max_price_minor BIGINT NOT NULL CHECK (max_price_minor > 0),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX subscriptions_active_direction_price_idx
    ON subscriptions (direction_from, direction_to, currency, max_price_minor)
    WHERE active;

CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    subscription_id BIGINT NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL,
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    actual_price_minor BIGINT NOT NULL CHECK (actual_price_minor > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (subscription_id, event_id)
);
