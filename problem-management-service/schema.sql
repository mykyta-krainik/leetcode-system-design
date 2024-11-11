CREATE TABLE problems (
  id SERIAL PRIMARY KEY,
  title VARCHAR(255),
  description TEXT,
  difficulty VARCHAR(50),
  tags TEXT[],
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
