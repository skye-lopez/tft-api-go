INSERT INTO matches (id) VALUES ($1) ON CONFLICT DO NOTHING;
