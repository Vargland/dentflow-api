package evolutions

import (
	"context"
	"errors"
	"math/big"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/psi-germanr/dentflow-api/internal/db/sqlc"
)

// ErrNotFound is returned when an evolution does not exist or does not belong to the doctor.
var ErrNotFound = errors.New("evolution not found")

// Repository handles database access for the evolutions module.
type Repository struct {
	q *db.Queries
}

// NewRepository creates a new evolutions Repository.
func NewRepository(pool *db.Pool) *Repository {
	return &Repository{q: db.New(pool)}
}

// List returns all evolutions for a patient scoped to the doctor.
func (r *Repository) List(ctx context.Context, patientID, doctorID string) ([]EvolutionResponse, error) {
	rows, err := r.q.ListEvolutions(ctx, patientID, doctorID)
	if err != nil {
		return nil, err
	}
	items := make([]EvolutionResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, toResponse(row))
	}
	return items, nil
}

// Create inserts a new evolution for a patient.
func (r *Repository) Create(ctx context.Context, patientID, doctorID string, req CreateEvolutionRequest) (EvolutionResponse, error) {
	dientes := req.Dientes
	if dientes == nil {
		dientes = []int32{}
	}

	e, err := r.q.CreateEvolution(ctx, db.CreateEvolutionParams{
		PatientID:   patientID,
		DoctorID:    doctorID,
		Descripcion: req.Descripcion,
		Dientes:     dientes,
		Importe:     float64ToNumeric(req.Importe),
		Pagado:      req.Pagado,
	})
	if err != nil {
		return EvolutionResponse{}, err
	}
	return toResponse(e), nil
}

// Update modifies an existing evolution.
func (r *Repository) Update(ctx context.Context, id, doctorID string, req UpdateEvolutionRequest) (EvolutionResponse, error) {
	var importePtr *pgtype.Numeric
	if req.Importe != nil {
		n := float64ToNumeric(req.Importe)
		importePtr = &n
	}

	e, err := r.q.UpdateEvolution(ctx, db.UpdateEvolutionParams{
		ID:          id,
		DoctorID:    doctorID,
		Descripcion: req.Descripcion,
		Dientes:     req.Dientes,
		Importe:     importePtr,
		Pagado:      req.Pagado,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return EvolutionResponse{}, ErrNotFound
	}
	if err != nil {
		return EvolutionResponse{}, err
	}
	return toResponse(e), nil
}

// Delete removes an evolution.
func (r *Repository) Delete(ctx context.Context, id, doctorID string) error {
	return r.q.DeleteEvolution(ctx, id, doctorID)
}

// toResponse converts a db.Evolution to the API shape.
func toResponse(e db.Evolution) EvolutionResponse {
	var importeF *float64
	if e.Importe.Valid {
		f, _ := e.Importe.Float64Value()
		if f.Valid {
			importeF = &f.Float64
		}
	}
	dientes := e.Dientes
	if dientes == nil {
		dientes = []int32{}
	}
	return EvolutionResponse{
		ID:          e.ID,
		PatientID:   e.PatientID,
		Descripcion: e.Descripcion,
		Dientes:     dientes,
		Importe:     importeF,
		Pagado:      e.Pagado,
		Fecha:       e.Fecha,
		CreatedAt:   e.CreatedAt,
	}
}

// float64ToNumeric converts a *float64 to pgtype.Numeric.
func float64ToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	_ = n.Scan(new(big.Float).SetFloat64(*f).Text('f', 2))
	return n
}
