package util

type Error struct {
	Detail string `json:"detail"`
	Status string `json:"status"`
}
type ErrorDocument struct {
	Errors []Error `json:"errors"`
}

func ErrorDoc(message, status string) *ErrorDocument {
	return &ErrorDocument{
		[]Error{{
			Detail: message,
			Status: status,
		}},
	}
}
