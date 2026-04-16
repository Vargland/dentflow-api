// Package email provides transactional email sending via Resend.
package email

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/resend/resend-go/v2"
)

// InviteParams holds all data needed to send an appointment invitation.
type InviteParams struct {
	// Patient info
	PatientName  string
	PatientEmail string

	// Doctor / clinic info
	DoctorName    string
	ClinicAddress string
	ClinicPhone   string

	// Appointment info
	Title     string
	StartTime time.Time // in doctor's local timezone
	EndTime   time.Time // in doctor's local timezone
	StartUTC  time.Time // UTC — for GCal link
	EndUTC    time.Time // UTC — for GCal link
	Duration  int       // minutes

	// Language: "es" | "en"
	Language string
}

// SendInvite sends an appointment invitation email to the patient.
// Returns an error if the Resend API call fails.
func SendInvite(p InviteParams) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("email.SendInvite: RESEND_API_KEY not set")
	}

	from := os.Getenv("RESEND_FROM")
	if from == "" {
		from = "onboarding@resend.dev"
	}

	subject := buildSubject(p)
	html := buildHTML(p)

	client := resend.NewClient(apiKey)

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{p.PatientEmail},
		Subject: subject,
		Html:    html,
	}

	_, err := client.Emails.Send(params)

	return err
}

// ---------- content builders ----------

func buildSubject(p InviteParams) string {
	dateStr := formatDate(p.StartTime, p.Language)

	if p.Language == "en" {
		return fmt.Sprintf("Appointment confirmed — %s", dateStr)
	}

	return fmt.Sprintf("Turno confirmado — %s", dateStr)
}

func buildHTML(p InviteParams) string {
	gcalURL := buildGCalURL(p)
	dateStr := formatDate(p.StartTime, p.Language)
	timeStr := fmt.Sprintf("%s – %s", p.StartTime.Format("15:04"), p.EndTime.Format("15:04"))

	var greeting, professional, address, phone, dateLabel, timeLabel, durationLabel, addToCalBtn, footer string

	if p.Language == "en" {
		greeting = fmt.Sprintf("Hello %s,", p.PatientName)
		professional = "Professional"
		address = "Address"
		phone = "Phone"
		dateLabel = "Date"
		timeLabel = "Time"
		durationLabel = "Duration"
		addToCalBtn = "Add to Google Calendar"
		footer = "DentFlow · Dental practice management"
	} else {
		greeting = fmt.Sprintf("Hola %s,", p.PatientName)
		professional = "Profesional"
		address = "Dirección"
		phone = "Teléfono"
		dateLabel = "Fecha"
		timeLabel = "Horario"
		durationLabel = "Duración"
		addToCalBtn = "Agregar a Google Calendar"
		footer = "DentFlow · Sistema de gestión odontológica"
	}

	var bodyIntro string
	if p.Language == "en" {
		bodyIntro = "Your appointment has been scheduled:"
	} else {
		bodyIntro = "Tu turno ha sido agendado:"
	}

	var rows strings.Builder

	rows.WriteString(tableRow(professional, p.DoctorName))
	if p.ClinicAddress != "" {
		rows.WriteString(tableRow(address, p.ClinicAddress))
	}
	if p.ClinicPhone != "" {
		rows.WriteString(tableRow(phone, p.ClinicPhone))
	}
	rows.WriteString(tableRow(dateLabel, dateStr))
	rows.WriteString(tableRow(timeLabel, timeStr))
	rows.WriteString(tableRow(durationLabel, fmt.Sprintf("%d min", p.Duration)))

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
</head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:40px 0;">
    <tr>
      <td align="center">
        <table width="560" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:12px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,0.08);">

          <!-- Header -->
          <tr>
            <td style="background:#2563eb;padding:28px 40px;">
              <p style="margin:0;font-size:22px;font-weight:700;color:#ffffff;">✓ %s</p>
            </td>
          </tr>

          <!-- Body -->
          <tr>
            <td style="padding:32px 40px;">
              <p style="margin:0 0 8px;font-size:15px;color:#374151;">%s</p>
              <p style="margin:0 0 24px;font-size:14px;color:#6b7280;">%s</p>

              <!-- Details table -->
              <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f9fafb;border:1px solid #e5e7eb;border-radius:8px;overflow:hidden;">
                %s
              </table>

              <!-- GCal button -->
              <table cellpadding="0" cellspacing="0" style="margin-top:28px;">
                <tr>
                  <td style="background:#2563eb;border-radius:8px;">
                    <a href="%s" target="_blank"
                       style="display:inline-block;padding:12px 24px;font-size:14px;font-weight:600;color:#ffffff;text-decoration:none;">
                      📅 %s
                    </a>
                  </td>
                </tr>
              </table>
            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="background:#f9fafb;padding:16px 40px;border-top:1px solid #e5e7eb;">
              <p style="margin:0;font-size:12px;color:#9ca3af;">%s</p>
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
		p.Language,
		buildSubject(p),
		buildSubject(p),
		greeting,
		bodyIntro,
		rows.String(),
		gcalURL,
		addToCalBtn,
		footer,
	)
}

func tableRow(label, value string) string {
	return fmt.Sprintf(`
  <tr>
    <td style="padding:10px 16px;font-size:13px;font-weight:600;color:#6b7280;white-space:nowrap;width:120px;border-bottom:1px solid #e5e7eb;">%s</td>
    <td style="padding:10px 16px;font-size:13px;color:#111827;border-bottom:1px solid #e5e7eb;">%s</td>
  </tr>`, label, value)
}

func buildGCalURL(p InviteParams) string {
	// Google Calendar uses compact UTC format: 20250418T130000Z
	const gcalFmt = "20060102T150405Z"

	startStr := p.StartUTC.UTC().Format(gcalFmt)
	endStr := p.EndUTC.UTC().Format(gcalFmt)

	details := fmt.Sprintf("Profesional: %s", p.DoctorName)
	if p.ClinicAddress != "" {
		details += "\nDirección: " + p.ClinicAddress
	}
	if p.ClinicPhone != "" {
		details += "\nTeléfono: " + p.ClinicPhone
	}

	params := url.Values{}
	params.Set("action", "TEMPLATE")
	params.Set("text", fmt.Sprintf("Turno con %s", p.DoctorName))
	params.Set("dates", startStr+"/"+endStr)
	params.Set("details", details)
	if p.ClinicAddress != "" {
		params.Set("location", p.ClinicAddress)
	}

	return "https://www.google.com/calendar/render?" + params.Encode()
}

// formatDate formats a time.Time for display in the given language.
func formatDate(t time.Time, lang string) string {
	if lang == "en" {
		return t.Format("Monday, January 2, 2006")
	}

	// Spanish — manual map for locale
	days := []string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}
	months := []string{"enero", "febrero", "marzo", "abril", "mayo", "junio",
		"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}

	return fmt.Sprintf("%s, %d de %s de %d",
		days[t.Weekday()],
		t.Day(),
		months[t.Month()-1],
		t.Year(),
	)
}
