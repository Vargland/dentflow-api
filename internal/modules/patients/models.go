// Package patients implements the patients HTTP module.
package patients

import (
	"encoding/json"
	"time"
)

// PatientResponse is the JSON shape returned to the frontend.
type PatientResponse struct {
	ID              string          `json:"id"`
	Nombre          string          `json:"nombre"`
	Apellido        string          `json:"apellido"`
	Dni             *string         `json:"dni"`
	FechaNacimiento *string         `json:"fechaNacimiento"`
	Sexo            *string         `json:"sexo"`
	Telefono        *string         `json:"telefono"`
	Email           *string         `json:"email"`
	Direccion       *string         `json:"direccion"`
	Alergias        *string         `json:"alergias"`
	Medicamentos    *string         `json:"medicamentos"`
	Antecedentes    *string         `json:"antecedentes"`
	ObraSocial      *string         `json:"obraSocial"`
	NroAfiliado     *string         `json:"nroAfiliado"`
	Notas           *string         `json:"notas"`
	Odontograma     json.RawMessage `json:"odontograma"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

// PatientListItem is the lighter shape used in the patient list.
type PatientListItem struct {
	ID             string    `json:"id"`
	Nombre         string    `json:"nombre"`
	Apellido       string    `json:"apellido"`
	Dni            *string   `json:"dni"`
	Telefono       *string   `json:"telefono"`
	ObraSocial     *string   `json:"obraSocial"`
	CreatedAt      time.Time `json:"createdAt"`
	EvolutionCount int64     `json:"evolutionCount"`
}

// CreatePatientRequest is the request body for POST /patients.
type CreatePatientRequest struct {
	Nombre          string  `json:"nombre"`
	Apellido        string  `json:"apellido"`
	Dni             *string `json:"dni"`
	FechaNacimiento *string `json:"fechaNacimiento"`
	Sexo            *string `json:"sexo"`
	Telefono        *string `json:"telefono"`
	Email           *string `json:"email"`
	Direccion       *string `json:"direccion"`
	Alergias        *string `json:"alergias"`
	Medicamentos    *string `json:"medicamentos"`
	Antecedentes    *string `json:"antecedentes"`
	ObraSocial      *string `json:"obraSocial"`
	NroAfiliado     *string `json:"nroAfiliado"`
	Notas           *string `json:"notas"`
}

// UpdatePatientRequest is the request body for PUT /patients/:id.
// All fields are optional — nil means "leave unchanged".
type UpdatePatientRequest struct {
	Nombre          *string `json:"nombre"`
	Apellido        *string `json:"apellido"`
	Dni             *string `json:"dni"`
	FechaNacimiento *string `json:"fechaNacimiento"`
	Sexo            *string `json:"sexo"`
	Telefono        *string `json:"telefono"`
	Email           *string `json:"email"`
	Direccion       *string `json:"direccion"`
	Alergias        *string `json:"alergias"`
	Medicamentos    *string `json:"medicamentos"`
	Antecedentes    *string `json:"antecedentes"`
	ObraSocial      *string `json:"obraSocial"`
	NroAfiliado     *string `json:"nroAfiliado"`
	Notas           *string `json:"notas"`
}

// SaveOdontogramRequest is the request body for PUT /patients/:id/odontogram.
type SaveOdontogramRequest struct {
	Data json.RawMessage `json:"data"`
}

// OdontogramResponse is returned after GET/PUT /patients/:id/odontogram.
type OdontogramResponse struct {
	PatientID   string          `json:"patientId"`
	Data        json.RawMessage `json:"data"`
	UpdatedAt   *time.Time      `json:"updatedAt"`
}
