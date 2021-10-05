package util

type Error struct {
	Detail string `json:"detail"`
	Status string `json:"status"`
}
type ErrorDocument struct {
	Errors []Error `json:"errors"`
}

func (e *Error) ErrorDocument(message, status string) *ErrorDocument {
	return &ErrorDocument{
		[]Error{{
			Detail: message,
			Status: status,
		}},
	}
}

type Logger interface {
	Error(i ...interface{})
}

type ErrorLog struct {
	LogMessage string
	Logger     Logger
	Message    string
	Status     string
}

func (e *ErrorLog) ErrorDocument() *ErrorDocument {
	logMessage := e.LogMessage
	if logMessage == "" {
		if e.Message == "" {
			logMessage = "Bad Request"
		} else {
			logMessage = e.Message
		}
	}

	if logMessage != "" && e.Logger != nil {
		e.Logger.Error(e.LogMessage)
	}

	status := e.Status
	if status == "" {
		status = "400"
	}

	return (&Error{}).ErrorDocument(e.Message, status)
}
