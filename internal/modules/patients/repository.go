package patients

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	db "github.com/psi-germanr/dentflow-api/internal/db/sqlc"
)

// ErrNotFound is returned when a patient is not found or does not belong to the doctor.
var ErrNotFound = errors.New("patient not found")

// Repository handles all database access for the patients module.
type Repository struct {
	q *db.Queries
}

// NewRepository creates a new patients Repository.
func NewRepository(pool *db.Pool) *Repository {
	return &Repository{q: db.New(pool)}
}

// List returns all patients for the given doctor, optionally filtered by query.
func (r *Repository) List(ctx context.Context, doctorID string, query *string) ([]PatientListItem, error) {
	rows, err := r.q.ListPatients(ctx, doctorID, query)
	if err != nil {
		return nil, err
	}

	items := make([]PatientListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, PatientListItem{
			ID:             row.ID,
			Nombre:         row.Nombre,
			Apellido:       row.Apellido,
			Dni:            row.Dni,
			Telefono:       row.Telefono,
			ObraSocial:     row.ObraSocial,
			CreatedAt:      row.CreatedAt,
			EvolutionCount: row.EvolutionCount,
		})
	}
	return items, nil
}

// Get returns a single patient by ID scoped to the doctor.
func (r *Repository) Get(ctx context.Context, id, doctorID string) (PatientResponse, error) {
	p, err := r.q.GetPatient(ctx, id, doctorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return PatientResponse{}, ErrNotFound
	}
	if err != nil {
		return PatientResponse{}, err
	}
	return toResponse(p), nil
}

// Create inserts a new patient for the doctor.
func (r *Repository) Create(ctx context.Context, doctorID string, req CreatePatientRequest) (PatientResponse, error) {
	var dob *time.Time
	if req.FechaNacimiento != nil && *req.FechaNacimiento != "" {
		t, err := time.Parse("2006-01-02", *req.FechaNacimiento)
		if err != nil {
			return PatientResponse{}, errors.New("invalid fechaNacimiento format, use YYYY-MM-DD")
		}
		dob = &t
	}

	p, err := r.q.CreatePatient(ctx, db.CreatePatientParams{
		DoctorID:        doctorID,
		Nombre:          req.Nombre,
		Apellido:        req.Apellido,
		Dni:             req.Dni,
		FechaNacimiento: dob,
		Sexo:            req.Sexo,
		Telefono:        req.Telefono,
		Email:           req.Email,
		Direccion:       req.Direccion,
		Alergias:        req.Alergias,
		Medicamentos:    req.Medicamentos,
		Antecedentes:    req.Antecedentes,
		ObraSocial:      req.ObraSocial,
		NroAfiliado:     req.NroAfiliado,
		Notas:           req.Notas,
	})
	if err != nil {
		return PatientResponse{}, err
	}
	return toResponse(p), nil
}

// Update modifies an existing patient's fields (nil = keep existing).
func (r *Repository) Update(ctx context.Context, id, doctorID string, req UpdatePatientRequest) (PatientResponse, error) {
	var dob *time.Time
	if req.FechaNacimiento != nil && *req.FechaNacimiento != "" {
		t, err := time.Parse("2006-01-02", *req.FechaNacimiento)
		if err != nil {
			return PatientResponse{}, errors.New("invalid fechaNacimiento format, use YYYY-MM-DD")
		}
		dob = &t
	}

	p, err := r.q.UpdatePatient(ctx, db.UpdatePatientParams{
		ID:              id,
		DoctorID:        doctorID,
		Nombre:          req.Nombre,
		Apellido:        req.Apellido,
		Dni:             req.Dni,
		FechaNacimiento: dob,
		Sexo:            req.Sexo,
		Telefono:        req.Telefono,
		Email:           req.Email,
		Direccion:       req.Direccion,
		Alergias:        req.Alergias,
		Medicamentos:    req.Medicamentos,
		Antecedentes:    req.Antecedentes,
		ObraSocial:      req.ObraSocial,
		NroAfiliado:     req.NroAfiliado,
		Notas:           req.Notas,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return PatientResponse{}, ErrNotFound
	}
	if err != nil {
		return PatientResponse{}, err
	}
	return toResponse(p), nil
}

// Delete removes a patient.
func (r *Repository) Delete(ctx context.Context, id, doctorID string) error {
	return r.q.DeletePatient(ctx, id, doctorID)
}

// SaveOdontogram persists the odontogram JSON.
func (r *Repository) SaveOdontogram(ctx context.Context, id, doctorID string, data json.RawMessage) (OdontogramResponse, error) {
	row, err := r.q.SaveOdontogram(ctx, id, doctorID, data)
	if errors.Is(err, pgx.ErrNoRows) {
		return OdontogramResponse{}, ErrNotFound
	}
	if err != nil {
		return OdontogramResponse{}, err
	}
	return OdontogramResponse{
		PatientID: row.ID,
		Data:      row.Odontograma,
		UpdatedAt: &row.UpdatedAt,
	}, nil
}

// GetOdontogram returns the current odontogram for a patient.
func (r *Repository) GetOdontogram(ctx context.Context, id, doctorID string) (OdontogramResponse, error) {
	p, err := r.q.GetPatient(ctx, id, doctorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return OdontogramResponse{}, ErrNotFound
	}
	if err != nil {
		return OdontogramResponse{}, err
	}
	return OdontogramResponse{
		PatientID: p.ID,
		Data:      p.Odontograma,
		UpdatedAt: &p.UpdatedAt,
	}, nil
}

// toResponse converts a db.Patient to the API PatientResponse shape.
func toResponse(p db.Patient) PatientResponse {
	var dobStr *string
	if p.FechaNacimiento != nil {
		s := p.FechaNacimiento.Format("2006-01-02")
		dobStr = &s
	}
	return PatientResponse{
		ID:              p.ID,
		Nombre:          p.Nombre,
		Apellido:        p.Apellido,
		Dni:             p.Dni,
		FechaNacimiento: dobStr,
		Sexo:            p.Sexo,
		Telefono:        p.Telefono,
		Email:           p.Email,
		Direccion:       p.Direccion,
		Alergias:        p.Alergias,
		Medicamentos:    p.Medicamentos,
		Antecedentes:    p.Antecedentes,
		ObraSocial:      p.ObraSocial,
		NroAfiliado:     p.NroAfiliado,
		Notas:           p.Notas,
		Odontograma:     p.Odontograma,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}
