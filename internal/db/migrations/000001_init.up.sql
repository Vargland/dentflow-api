-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Patients
CREATE TABLE IF NOT EXISTS patients (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    doctor_id     TEXT NOT NULL,
    nombre        TEXT NOT NULL,
    apellido      TEXT NOT NULL,
    dni           TEXT,
    fecha_nacimiento DATE,
    sexo          TEXT,
    telefono      TEXT,
    email         TEXT,
    direccion     TEXT,
    alergias      TEXT,
    medicamentos  TEXT,
    antecedentes  TEXT,
    obra_social   TEXT,
    nro_afiliado  TEXT,
    notas         TEXT,
    odontograma   JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_patients_doctor_id ON patients(doctor_id);
CREATE INDEX IF NOT EXISTS idx_patients_apellido   ON patients(doctor_id, apellido);

-- Evolutions (clinical records)
CREATE TABLE IF NOT EXISTS evolutions (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    patient_id  TEXT NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    doctor_id   TEXT NOT NULL,
    descripcion TEXT NOT NULL,
    dientes     INTEGER[] NOT NULL DEFAULT '{}',
    importe     NUMERIC(12,2),
    pagado      BOOLEAN NOT NULL DEFAULT FALSE,
    fecha       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_evolutions_patient_id ON evolutions(patient_id);
CREATE INDEX IF NOT EXISTS idx_evolutions_doctor_id  ON evolutions(doctor_id);

-- Auto-update updated_at on patients
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER patients_updated_at
    BEFORE UPDATE ON patients
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
