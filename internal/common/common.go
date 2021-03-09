package common

import "time"

// CatalogConfig stores the config parameters for the
// Catalog Worker
type CatalogConfig struct {
	Level                 string // The Log Level
	URL                   string // The URL to your Ansible Tower
	Token                 string // The Token used to authenticate with Ansible Tower
	SkipVerifyCertificate bool   // Skip Certifcate Validation
	MQTTURL               string // The URL for MQTT Server
	GUID                  string // The Client GUID
}

// JobParam stores the single parameter set for a job
type JobParam struct {
	Method                 string                 `json:"method"`
	HrefSlug               string                 `json:"href_slug"`
	FetchAllPages          bool                   `json:"fetch_all_pages"`
	Params                 map[string]interface{} `json:"params"`
	ApplyFilter            interface{}            `json:"apply_filter"`
	RefreshIntervalSeconds int64                  `json:"refresh_interval_seconds"`
	FetchRelated           []interface{}          `json:"fetch_related"`
	PagePrefix             string                 `json:"page_prefix"`
}

// RequestInput describes the struct of input attribute in RequestMessage
type RequestInput struct {
	ResponseFormat string     `json:"response_format"`
	UploadURL      string     `json:"upload_url"`
	Jobs           []JobParam `json:"jobs"`
	PreviousSHA    string     `json:"previous_sha"`
	PreviousSize   int64      `json:"previous_size"`
}

// CatalogInventoryTask stores all attributes of a task retrived from catalog-inventory API
type CatalogInventoryTask struct {
	Input     RequestInput `json:"input"`
	CreatedAt time.Time    `json:"created_at"`
	ID        string       `json:"id"`
	State     string       `json:"state"`
	Status    string       `json:"status"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// MQTTMessage stores all attributes of the MQTTMessage sent by catalog-inventory API
// TODO: remove when mqtt client is no longer needed
type MQTTMessage struct {
	URL string `json:"url"`
}

// Page stores data in a page with a name
type Page struct {
	Data []byte
	Name string
}

// PageWriter is an interface that writes named page contents and flushes
type PageWriter interface {
	Write(name string, b []byte) error
	Flush() error
	FlushErrors(msg []string) error
}
