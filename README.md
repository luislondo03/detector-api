# Driver Vision API

API del piloto empresarial para terminales Driver Vision. Conserva los endpoints de demo de incidentes por compatibilidad, pero el flujo nuevo enrola terminales con un QR temporal y exige credencial revocable de dispositivo.

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
- `POST /api/v1/telemetry`: recibe un punto de flota de una sesión activa.
- `GET /api/v1/fleet/latest`: devuelve la última posición recibida por cada dispositivo (hasta 50 por defecto).
- `POST /api/v1/device-enrollment/redeem`: canjea el código temporal del QR y entrega la credencial del terminal una sola vez.
- `POST /api/v1/device/{events,sessions,coverage,telemetry}/batch` y `POST /api/v1/device/heartbeat`: ingestión autenticada, idempotente y con ACK por mensaje.
- `GET /api/v1/fleet/vehicles`: proyección de vehículos con último contacto, estado y ubicación.
- `GET /api/v1/fleet/events`: eventos de detección de terminales enrolados (tabla `detection_events`); admite `type`, `severity`, `vehicleId`, `from`, `to`, `limit` y `offset`.

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
  "details": {
    "source": "android",
    "monitoringContext": {
      "capturedAt": "2026-10-11T12:00:02Z",
      "location": {
        "latitude": 4.60971,
        "longitude": -74.08175,
        "accuracyMeters": 12,
        "speedKilometersPerHour": 34,
        "provider": "gps"
      },
      "device": { "manufacturer": "realme", "model": "RMX…", "androidVersion": "15", "appVersion": "0.1.0" },
      "battery": { "percent": 80, "charging": false }
    }
  },
  "alertTriggered": true
}
```

`details.monitoringContext` es opcional y solo debe enviarse con consentimiento explícito del usuario. En actualizaciones del mismo `eventId`, el repositorio combina las claves JSON para conservar el contexto capturado al inicio del incidente.

## Enrolamiento y telemetría de flota

Un operador crea empresa, vehículo, terminal y código de enrolamiento con la cabecera `X-Admin-Bootstrap-Key`. La respuesta devuelve `qrPayload`; conviértelo en un QR sin agregar datos adicionales. El terminal lo escanea y canjea el código una sola vez; la API guarda únicamente hashes del código y de la credencial resultante. La app Android cifra la credencial con Android Keystore. Durante el piloto, la pantalla web de aprovisionamiento permite que el operador introduzca esa clave manualmente en memoria para llamar estas rutas; la clave no se distribuye en el bundle ni se persiste. Esta excepción debe eliminarse antes de producción mediante autenticación humana y roles.

El servicio Android de ubicación envía telemetría cada minuto aun cuando la interfaz está minimizada, mediante una notificación persistente. La API elimina telemetría anterior a 90 días al arrancar y cada 24 horas; `TELEMETRY_RETENTION_DAYS=0` deshabilita esa limpieza. No se usan IMEI, número telefónico ni identificadores de hardware.

Para producción define retención/anonimización de telemetría, restringe acceso con autenticación y roles, y comunica claramente la política laboral y de privacidad. El demo no implementa estos controles ni rastreo en segundo plano.

## Variables de entorno

| Variable | Requerida | Propósito |
| --- | --- | --- |
| `DATABASE_URL` | Sí | URL de PostgreSQL. |
| `PORT` | No | Puerto HTTP, `8080` por defecto. |
| `EVIDENCE_DIRECTORY` | No | Directorio de evidencias, `./storage/evidence` por defecto. |
| `ALLOWED_ORIGIN` | No | Origen CORS del dashboard; use su URL de Railway en producción. |
| `ADMIN_BOOTSTRAP_KEY` | Sí para administración | Secreto para crear empresas, vehículos, terminales, QR y revocaciones durante el piloto. La pantalla piloto lo recibe solo mediante ingreso manual y nunca lo persiste o distribuye en el bundle. |
| `TELEMETRY_RETENTION_DAYS` | No | Días de conservación de ubicación; `90` por defecto, `0` deshabilita la limpieza. |

## Docker y Railway

El `Dockerfile` está listo para desplegar el servicio. Railway puede asignar el puerto con `PORT` y la base PostgreSQL con `DATABASE_URL`.

Las fotos se guardan en el sistema de archivos del contenedor. Ese almacenamiento es efímero al redeploy/reinicio salvo que asignes un volumen persistente y apuntes `EVIDENCE_DIRECTORY` a su punto de montaje (por ejemplo, `/data/evidence`). Para una demo sin volumen se aceptan esa pérdida y la retención limitada; no es una solución de producción.

El dashboard aún no tiene login por decisión de piloto. Las rutas administrativas sí requieren `ADMIN_BOOTSTRAP_KEY` y las rutas `/device/*` requieren una credencial revocable por terminal. Antes de producción deben incorporarse autenticación humana, roles, auditoría ampliada y un scheduler de retención garantizada.
