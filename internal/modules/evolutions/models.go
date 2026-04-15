// Package evolutions implements the clinical records HTTP module.
package evolutions

import "time"

// EvolutionResponse is the JSON shape returned to the frontend.
type EvolutionResponse struct {
	ID          string    `json:"id"`
	PatientID   string    `json:"patientId"`
	Descripcion string    `json:"descripcion"`
	Dientes     []int32   `json:"dientes"`
	Importe     *float64  `json:"importe"`
	Pagado      bool      `json:"pagado"`
	Fecha       time.Time `json:"fecha"`
	CreatedAt   time.Time `json:"createdAt"`
}

// CreateEvolutionRequest is the request body for POST /patients/:id/evolutions.
type CreateEvolutionRequest struct {
	Descripcion string   `json:"descripcion"`
	Dientes     []int32  `json:"dientes"`
	Importe     *float64 `json:"importe"`
	Pagado      bool     `json:"pagado"`
}

// UpdateEvolutionRequest is the request body for PUT /patients/:id/evolutions/:eid.
type UpdateEvolutionRequest struct {
	Descripcion *string  `json:"descripcion"`
	Dientes     []int32  `json:"dientes"`
	Importe     *float64 `json:"importe"`
	Pagado      *bool    `json:"pagado"`
}
