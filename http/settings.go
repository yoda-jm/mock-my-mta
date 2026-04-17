package http

import (
	"encoding/json"
	"net/http"
	"sync"
)

// SmtpSettings are runtime-configurable SMTP behavior settings.
// These can be changed via the settings modal or API without restarting.
type SmtpSettings struct {
	// RejectRate is the percentage (0-100) of emails to reject with a 5xx error.
	RejectRate int `json:"reject_rate"`
	// RejectMessage is the error message returned when rejecting.
	RejectMessage string `json:"reject_message"`
	// DelayMs is the delay in milliseconds before accepting each email.
	DelayMs int `json:"delay_ms"`
	// BounceRate is the percentage (0-100) of emails to accept then bounce.
	BounceRate int `json:"bounce_rate"`
	// BounceMessage is the DSN message for bounced emails.
	BounceMessage string `json:"bounce_message"`
}

var (
	smtpSettings     = SmtpSettings{
		RejectMessage: "550 Mailbox unavailable (mock rejection)",
		BounceMessage: "Your message could not be delivered (mock bounce)",
	}
	smtpSettingsMu sync.RWMutex
)

// GetSmtpSettings returns the current SMTP behavior settings.
func GetSmtpSettings() SmtpSettings {
	smtpSettingsMu.RLock()
	defer smtpSettingsMu.RUnlock()
	return smtpSettings
}

func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	smtpSettingsMu.RLock()
	defer smtpSettingsMu.RUnlock()
	writeJSONResponse(w, smtpSettings)
}

func handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var newSettings SmtpSettings
	if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid settings: %v", err)
		return
	}
	// Clamp values
	if newSettings.RejectRate < 0 {
		newSettings.RejectRate = 0
	}
	if newSettings.RejectRate > 100 {
		newSettings.RejectRate = 100
	}
	if newSettings.DelayMs < 0 {
		newSettings.DelayMs = 0
	}
	if newSettings.BounceRate < 0 {
		newSettings.BounceRate = 0
	}
	if newSettings.BounceRate > 100 {
		newSettings.BounceRate = 100
	}
	if newSettings.RejectMessage == "" {
		newSettings.RejectMessage = "550 Mailbox unavailable (mock rejection)"
	}
	if newSettings.BounceMessage == "" {
		newSettings.BounceMessage = "Your message could not be delivered (mock bounce)"
	}

	smtpSettingsMu.Lock()
	smtpSettings = newSettings
	smtpSettingsMu.Unlock()

	BroadcastEvent("settings_changed", newSettings)
	writeJSONResponse(w, newSettings)
}
