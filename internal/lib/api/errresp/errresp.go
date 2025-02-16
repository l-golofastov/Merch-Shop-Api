package errresp

type ErrorResponse struct {
	Error string `json:"errors"`
}

func Error(msg string) ErrorResponse {
	return ErrorResponse{msg}
}
