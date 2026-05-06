-- name: ListEvolutions :many
SELECT * FROM evolutions
WHERE patient_id = $1 AND doctor_id = $2
ORDER BY fecha DESC;

-- name: GetEvolution :one
SELECT * FROM evolutions
WHERE id = $1 AND doctor_id = $2
LIMIT 1;

-- name: CreateEvolution :one
INSERT INTO evolutions (
    patient_id, doctor_id, descripcion, dientes, importe, pagado, fecha
) VALUES (
    $1, $2, $3, $4, $5, $6, COALESCE($7, NOW())
)
RETURNING id, patient_id, doctor_id, descripcion, dientes, importe, pagado, fecha, created_at;

-- name: UpdateEvolution :one
UPDATE evolutions SET
    descripcion = COALESCE(sqlc.narg('descripcion'), descripcion),
    dientes     = COALESCE(sqlc.narg('dientes'),     dientes),
    importe     = COALESCE(sqlc.narg('importe'),     importe),
    pagado      = COALESCE(sqlc.narg('pagado'),      pagado),
    fecha       = COALESCE(sqlc.narg('fecha'),       fecha)
WHERE id = sqlc.arg('id') AND doctor_id = sqlc.arg('doctor_id')
RETURNING id, patient_id, doctor_id, descripcion, dientes, importe, pagado, fecha, created_at;

-- name: DeleteEvolution :exec
DELETE FROM evolutions
WHERE id = $1 AND doctor_id = $2;
