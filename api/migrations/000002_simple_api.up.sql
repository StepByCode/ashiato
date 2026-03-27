CREATE TABLE simple_tasks (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  owner TEXT NOT NULL CHECK (owner IN ('kido', 'kitahara', 'sogo', 'nakai')),
  state TEXT NOT NULL DEFAULT 'in_progress' CHECK (state IN ('in_progress', 'done', 'approved')),
  url TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE meeting_settings (
  id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  meeting_at TIMESTAMPTZ,
  meet_url TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO meeting_settings (id) VALUES (1);

CREATE TABLE publicity_template (
  id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  text TEXT NOT NULL DEFAULT '' CHECK (char_length(text) <= 140),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO publicity_template (id) VALUES (1);

CREATE TABLE publicity_channels (
  id TEXT PRIMARY KEY CHECK (id IN ('x', 'instagram', 'facebook')),
  name TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT 'in_progress' CHECK (state IN ('in_progress', 'done')),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO publicity_channels (id, name, note) VALUES
  ('x', 'X', ''),
  ('instagram', 'Instagram', ''),
  ('facebook', 'Facebook', '');
