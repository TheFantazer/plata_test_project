CREATE TABLE quote_updates
(
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    base_currency  VARCHAR(3)  NOT NULL,
    quote_currency VARCHAR(3)  NOT NULL,
    status         TEXT        NOT NULL DEFAULT 'pending',
    price          NUMERIC(20, 10),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fetched_at     TIMESTAMPTZ,
    error_message  TEXT,

    CONSTRAINT quote_updates_base_currency_check
        CHECK (base_currency IN ('USD', 'EUR', 'MXN')),
    CONSTRAINT quote_updates_quote_currency_check
        CHECK (quote_currency IN ('USD', 'EUR', 'MXN')),
    CONSTRAINT quote_updates_pair_check
        CHECK (base_currency <> quote_currency),
    CONSTRAINT quote_updates_status_check
        CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    CONSTRAINT quote_updates_price_check
        CHECK (price IS NULL OR price > 0),
    CONSTRAINT quote_updates_completed_check
        CHECK (status <> 'completed' OR (price IS NOT NULL AND fetched_at IS NOT NULL))
);

CREATE INDEX quote_updates_latest_idx
    ON quote_updates (base_currency, quote_currency, fetched_at DESC, id DESC)
    WHERE status='completed';

CREATE UNIQUE INDEX quote_updates_one_active_pair_idx
    ON quote_updates (base_currency, quote_currency)
    WHERE status IN ('pending', 'processing');