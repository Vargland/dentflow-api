CREATE TABLE appointments (
    id               TEXT        NOT NULL DEFAULT gen_random_uuid()::text PRIMARY KEY,
    doctor_id        TEXT        NOT NULL,
    patient_id       TEXT        REFERENCES patients(id) ON DELETE SET NULL,
    google_event_id  TEXT,
    title            TEXT        NOT NULL,
    start_time       TIMESTAMPTZ NOT NULL,
    end_time         TIMESTAMPTZ NOT NULL,
    duration_minutes INTEGER     NOT NULL DEFAULT 30,
    status           TEXT        NOT NULL DEFAULT 'scheduled',
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_appointments_doctor_id  ON appointments (doctor_id);
CREATE INDEX idx_appointments_start_time ON appointments (doctor_id, start_time);

CREATE TRIGGER appointments_updated_at
    BEFORE UPDATE ON appointments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
