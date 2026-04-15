-- name: ListPatients :many
SELECT
    id,
    nombre,
    apellido,
    dni,
    telefono,
    obra_social,
    created_at,
    (SELECT COUNT(*) FROM evolutions e WHERE e.patient_id = p.id) AS evolution_count
FROM patients p
WHERE doctor_id = $1
  AND (
    $2::text IS NULL
    OR nombre    ILIKE '%' || $2 || '%'
    OR apellido  ILIKE '%' || $2 || '%'
    OR dni       ILIKE '%' || $2 || '%'
  )
ORDER BY apellido ASC;

-- name: GetPatient :one
SELECT * FROM patients
WHERE id = $1 AND doctor_id = $2
LIMIT 1;

-- name: CreatePatient :one
INSERT INTO patients (
    doctor_id, nombre, apellido, dni, fecha_nacimiento, sexo,
    telefono, email, direccion, alergias, medicamentos,
    antecedentes, obra_social, nro_afiliado, notas
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14, $15
)
RETURNING *;

-- name: UpdatePatient :one
UPDATE patients SET
    nombre           = COALESCE(sqlc.narg('nombre'),           nombre),
    apellido         = COALESCE(sqlc.narg('apellido'),         apellido),
    dni              = COALESCE(sqlc.narg('dni'),              dni),
    fecha_nacimiento = COALESCE(sqlc.narg('fecha_nacimiento'), fecha_nacimiento),
    sexo             = COALESCE(sqlc.narg('sexo'),             sexo),
    telefono         = COALESCE(sqlc.narg('telefono'),         telefono),
    email            = COALESCE(sqlc.narg('email'),            email),
    direccion        = COALESCE(sqlc.narg('direccion'),        direccion),
    alergias         = COALESCE(sqlc.narg('alergias'),         alergias),
    medicamentos     = COALESCE(sqlc.narg('medicamentos'),     medicamentos),
    antecedentes     = COALESCE(sqlc.narg('antecedentes'),     antecedentes),
    obra_social      = COALESCE(sqlc.narg('obra_social'),      obra_social),
    nro_afiliado     = COALESCE(sqlc.narg('nro_afiliado'),     nro_afiliado),
    notas            = COALESCE(sqlc.narg('notas'),            notas)
WHERE id = sqlc.arg('id') AND doctor_id = sqlc.arg('doctor_id')
RETURNING *;

-- name: DeletePatient :exec
DELETE FROM patients
WHERE id = $1 AND doctor_id = $2;

-- name: SaveOdontogram :one
UPDATE patients SET odontograma = $3
WHERE id = $1 AND doctor_id = $2
RETURNING id, odontograma, updated_at;
