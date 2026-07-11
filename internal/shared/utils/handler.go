package utils

import "net/http"

// Handle writes a standard 200 success response, or maps the error to the
// unified error response envelope.
func Handle(w http.ResponseWriter, data interface{}, err error) {
	if err != nil {
		MapError(w, err)
		return
	}
	RespondSuccess(w, data)
}

// HandleCreated writes a standard 201 created response, or maps the error to
// the unified error response envelope.
func HandleCreated(w http.ResponseWriter, data interface{}, err error) {
	if err != nil {
		MapError(w, err)
		return
	}
	RespondCreated(w, data)
}

// HandleNoContent writes a standard 200 success response with nil data, or
// maps the error to the unified error response envelope.
func HandleNoContent(w http.ResponseWriter, err error) {
	if err != nil {
		MapError(w, err)
		return
	}
	RespondSuccess(w, nil)
}

// HandlePaginated writes a standard 200 paginated response, or maps the error
// to the unified error response envelope.
func HandlePaginated(w http.ResponseWriter, data interface{}, page, perPage, total int, err error) {
	if err != nil {
		MapError(w, err)
		return
	}
	RespondPaginated(w, data, page, perPage, total)
}
