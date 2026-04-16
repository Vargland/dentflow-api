ALTER TABLE user_settings
    DROP COLUMN IF EXISTS doctor_name,
    DROP COLUMN IF EXISTS clinic_address,
    DROP COLUMN IF EXISTS clinic_phone,
    DROP COLUMN IF EXISTS email_language;
