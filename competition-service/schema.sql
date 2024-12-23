CREATE TABLE competitions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    problem_ids INT[],
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE outbox (
    id SERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    processed BOOLEAN DEFAULT FALSE,
    retries INT DEFAULT 0
);

CREATE TABLE inbox (
    id SERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    processed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    retries INT DEFAULT 0
);
