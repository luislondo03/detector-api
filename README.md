# Driver Vision API

API del demo para recibir incidentes confirmados y una evidencia fotográfica consentida por cada evento. No incorpora autenticación deliberadamente: debe desplegarse solo en un entorno de demostración controlado.

## Ejecutar localmente

1. Cree una base PostgreSQL y copie `.env.example` a un archivo de entorno local.
2. Defina `DATABASE_URL` y, opcionalmente, `EVIDENCE_DIRECTORY`.
3. Ejecute:

```powershell
go run ./cmd/api
```

La API queda en `http://localhost:8080`; el esquema se crea al arrancar.

## Contrato mínimo

- `POST /api/v1/incidents`: crea o actualiza un incidente por `eventId`.
- `GET /api/v1/incidents`: devuelve incidentes paginados; admite `type`, `from`, `to`, `limit` y `offset`.
- `GET /api/v1/incidents/{eventId}`: devuelve el detalle con evidencias.
- `POST /api/v1/incidents/{eventId}/evidence`: formulario multipart con `file` y `consent=true`. Solo JPEG/PNG de hasta 5 MiB; se inspecciona el contenido real.
- `GET /evidence/{archivo}`: sirve una evidencia almacenada localmente.

Ejemplo de incidente:

```json
{
  "eventId": "d4d80f43-2d0e-4de2-9ad4-ae4ed7c92d11",
  "sessionId": "ac2b7e85-67d6-4f95-8974-b886b2df9a7f",
  "type": "NODDING_OFF",
  "lifecycle": "STARTED",
  "severity": "CRITICAL",
  "startedAt": "2026-10-11T12:00:00Z",
  "confidence": 0.82,
  "observationQuality": 0.91,
  "detectorVersion": "android-mvp-1",
  "modelVersion": "face-landmarker-and-yolo",
  "thresholdProfileId": "default-mvp",
  "details": { "source": "android" },
  "alertTriggered": true
}
```

## Variables de entorno

| Variable | Requerida | Propósito |
| --- | --- | --- |
| `DATABASE_URL` | Sí | URL de PostgreSQL. |
| `PORT` | No | Puerto HTTP, `8080` por defecto. |
| `EVIDENCE_DIRECTORY` | No | Directorio de evidencias, `./storage/evidence` por defecto. |
| `ALLOWED_ORIGIN` | No | Origen CORS del dashboard; use su URL de Railway en producción. |

## Docker y Railway

El `Dockerfile` está listo para desplegar el servicio. Railway puede asignar el puerto con `PORT` y la base PostgreSQL con `DATABASE_URL`.

Las fotos se guardan en el sistema de archivos del contenedor. Ese almacenamiento es efímero al redeploy/reinicio salvo que asignes un volumen persistente y apuntes `EVIDENCE_DIRECTORY` a su punto de montaje (por ejemplo, `/data/evidence`). Para una demo sin volumen se aceptan esa pérdida y la retención limitada; no es una solución de producción.

La API y `/evidence` no tienen login. No expongas el enlace público con información sensible fuera del contexto de demostración.
