CREATE TABLE competitions (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255),
  description TEXT,
  problem_ids INT[],
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
