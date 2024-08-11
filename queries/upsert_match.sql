INSERT INTO matches (id, set, patch) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;
