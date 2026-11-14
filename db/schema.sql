-- Esquema de SceneTrack (Postgres / Supabase)

CREATE TABLE tracks (
    id     BIGSERIAL PRIMARY KEY,
    title  TEXT NOT NULL,
    artist TEXT NOT NULL,
    key    TEXT NOT NULL UNIQUE  -- clave normalizada: norm_key(artist, title)
);

CREATE TABLE works (
    id    BIGSERIAL PRIMARY KEY,
    kind  TEXT NOT NULL CHECK (kind IN ('movie', 'series', 'trailer')),
    title TEXT NOT NULL,
    year  INT
);
CREATE UNIQUE INDEX works_unique ON works (kind, lower(title), COALESCE(year, 0));

CREATE TABLE usages (
    id         BIGSERIAL PRIMARY KEY,
    track_id   BIGINT NOT NULL REFERENCES tracks(id),
    work_id    BIGINT NOT NULL REFERENCES works(id),
    season     INT,
    episode    INT,
    note       TEXT,
    status     TEXT NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX usages_unique
    ON usages (track_id, work_id, COALESCE(season, 0), COALESCE(episode, 0));
CREATE INDEX idx_usages_track ON usages(track_id) WHERE status = 'approved';

-- Seguridad: cierra el acceso público por la API REST de Supabase.
-- El backend en Go se conecta directo a Postgres, así que no le afecta.
ALTER TABLE tracks ENABLE ROW LEVEL SECURITY;
ALTER TABLE works  ENABLE ROW LEVEL SECURITY;
ALTER TABLE usages ENABLE ROW LEVEL SECURITY;

-- Normalización de artista y título (misma regla al cargar y al buscar).
CREATE OR REPLACE FUNCTION public.norm_text(s text)
RETURNS text
LANGUAGE sql IMMUTABLE
SET search_path = ''
AS $$
  SELECT btrim(regexp_replace(
           regexp_replace(
             regexp_replace(
               regexp_replace(
                 replace(
                   translate(lower(s), 'áàäâéèëêíìïîóòöôúùüûñç', 'aaaaeeeeiiiioooouuuunc'),
                   '&', ' and '),
                 '(\s*[(\[][^)\]]*[)\]])+\s*$', ''),
               '\s+-\s+.*$', ''),
             '[''’‘`´".,!?:;¡¿*]', '', 'g'),
           '\s+', ' ', 'g'));
$$;

CREATE OR REPLACE FUNCTION public.norm_key(artist text, title text)
RETURNS text
LANGUAGE sql IMMUTABLE
SET search_path = ''
AS $$
  SELECT regexp_replace(
           public.norm_text(regexp_replace(artist, '\s+(feat\.?|ft\.?|featuring)\s+.*$', '', 'i')),
           '^the ', '')
         || '|' || public.norm_text(title);
$$;

REVOKE EXECUTE ON FUNCTION public.norm_text(text), public.norm_key(text, text)
  FROM PUBLIC, anon, authenticated;

-- Carga una escena: crea la canción y la obra si no existen y las vincula.
CREATE OR REPLACE FUNCTION add_scene(
  p_artist text, p_title text, p_kind text, p_work text, p_year int,
  p_season int DEFAULT NULL, p_episode int DEFAULT NULL, p_note text DEFAULT NULL
) RETURNS void LANGUAGE sql AS $$
  WITH t AS (
    INSERT INTO tracks (title, artist, key)
    VALUES (p_title, p_artist, public.norm_key(p_artist, p_title))
    ON CONFLICT (key) DO UPDATE SET key = EXCLUDED.key
    RETURNING id
  ), w AS (
    INSERT INTO works (kind, title, year)
    VALUES (p_kind, p_work, p_year)
    ON CONFLICT (kind, lower(title), COALESCE(year, 0)) DO UPDATE SET kind = EXCLUDED.kind
    RETURNING id
  )
  INSERT INTO usages (track_id, work_id, season, episode, note, status)
  SELECT t.id, w.id, p_season, p_episode, p_note, 'approved' FROM t, w
  ON CONFLICT DO NOTHING;
$$;

REVOKE EXECUTE ON FUNCTION add_scene(text, text, text, text, int, int, int, text)
  FROM PUBLIC, anon, authenticated;

  CREATE TABLE submissions (
    id         BIGSERIAL PRIMARY KEY,
    artist     TEXT NOT NULL CHECK (length(artist) <= 200),
    title      TEXT NOT NULL CHECK (length(title) <= 200),
    kind       TEXT NOT NULL CHECK (kind IN ('movie', 'series', 'trailer')),
    work       TEXT NOT NULL CHECK (length(work) <= 200),
    year       INT  CHECK (year BETWEEN 1888 AND 2100),
    season     INT  CHECK (season > 0),
    episode    INT  CHECK (episode > 0),
    note       TEXT CHECK (length(note) <= 500),
    status     TEXT NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE submissions ENABLE ROW LEVEL SECURITY;

-- Aprobar un envío: lo carga como escena y lo marca como aprobado.
CREATE OR REPLACE FUNCTION approve_submission(p_id bigint)
RETURNS void LANGUAGE sql AS $$
  SELECT add_scene(artist, title, kind, work, year, season, episode, note)
  FROM submissions WHERE id = p_id AND status = 'pending';
  UPDATE submissions SET status = 'approved' WHERE id = p_id AND status = 'pending';
$$;

REVOKE EXECUTE ON FUNCTION approve_submission(bigint) FROM PUBLIC, anon, authenticated;