-- name: ListAppointments :many
SELECT a.*, p.nombre, p.apellido
FROM appointments a
LEFT JOIN patients p ON p.id = a.patient_id
WHERE a.doctor_id = $1
  AND a.start_time >= $2
  AND a.start_time <  $3
ORDER BY a.start_time ASC;

-- name: GetAppointment :one
SELECT a.*, p.nombre, p.apellido
FROM appointments a
LEFT JOIN patients p ON p.id = a.patient_id
WHERE a.id = $1 AND a.doctor_id = $2
LIMIT 1;

-- name: CreateAppointment :one
INSERT INTO appointments (doctor_id, patient_id, google_event_id, title, start_time, end_time, duration_minutes, status, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateAppointment :one
UPDATE appointments
SET title            = $3,
    patient_id       = $4,
    google_event_id  = $5,
    start_time       = $6,
    end_time         = $7,
    duration_minutes = $8,
    status           = $9,
    notes            = $10
WHERE id = $1 AND doctor_id = $2
RETURNING *;

-- name: DeleteAppointment :exec
DELETE FROM appointments WHERE id = $1 AND doctor_id = $2;

-- name: CountOverlapping :one
SELECT COUNT(*) FROM appointments
WHERE doctor_id = $1
  AND id != $2
  AND start_time < $4
  AND end_time   > $3;
