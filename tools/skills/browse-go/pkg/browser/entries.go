package browser

// ConsoleEntry represents a browser console log message.
type ConsoleEntry struct {
	Timestamp int64  `json:"ts"`
	Level     string `json:"level"` // log, warn, error, info, debug
	Text      string `json:"text"`
}

// NetworkEntry represents a single network request/response pair.
type NetworkEntry struct {
	Timestamp int64  `json:"ts"`
	Method    string `json:"method"`
	URL       string `json:"url"`
	Status    int    `json:"status,omitempty"`
	Duration  int64  `json:"duration,omitempty"`
	Size      int    `json:"size,omitempty"`
}

// DialogEntry represents a JavaScript dialog event.
type DialogEntry struct {
	Timestamp  int64  `json:"ts"`
	Type       string `json:"type"`       // alert, confirm, prompt, beforeunload
	Message    string `json:"message"`
	DefaultVal string `json:"default,omitempty"`
	Action     string `json:"action"`     // accepted, dismissed
	Response   string `json:"response,omitempty"`
}
