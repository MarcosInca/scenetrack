# SceneTrack

Escucha una canción con el micrófono y te dice en qué película o serie suena.

## Cómo funciona

Navegador → `POST /v1/identify` (audio) → AudD reconoce la canción → Postgres dice dónde suena → JSON.

## Requisitos

- Go 1.22 o superior
- Un Postgres (probado con Supabase)
- Un token de [AudD](https://dashboard.audd.io)

## Correrlo

1. Crear las tablas: ejecutar `db/schema.sql` en la base.
2. Copiar `.env.example` a `.env` y completar `DATABASE_URL` y `AUDD_API_TOKEN`.
3. `./run.sh`
4. Abrir http://localhost:8080

## Estructura

```
cmd/api                  arranque y conexiones
internal/domain          tipos y errores
internal/recognition     reconocedor de audio (AudD o mock)
internal/repository      consultas a Postgres
internal/service         caso de uso: reconocer y buscar usos
internal/transport       API HTTP
web                      página con el botón de grabar
db                       esquema de la base
```