package utils

import "net/http"

// Handle writes a standard 200 success response, or maps the error to the
// unified error response envelope.
func Handle(w http.ResponseWriter, r *http.Request, data interface{}, err error) {
	if err != nil {
		MapErrorFromRequest(w, r, err)
		return
	}
	RespondSuccess(w, data)
}

// HandleCreated writes a standard 201 created response, or maps the error to
// the unified error response envelope.
func HandleCreated(w http.ResponseWriter, r *http.Request, data interface{}, err error) {
	if err != nil {
		MapErrorFromRequest(w, r, err)
		return
	}
	RespondCreated(w, data)
}

// HandleNoContent writes a standard 200 success response with nil data, or
// maps the error to the unified error response envelope.
func HandleNoContent(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		MapErrorFromRequest(w, r, err)
		return
	}
	RespondSuccess(w, nil)
}

// HandlePaginated writes a standard 200 paginated response, or maps the error
// to the unified error response envelope.
func HandlePaginated(w http.ResponseWriter, r *http.Request, data interface{}, page, perPage, total int, err error) {
	if err != nil {
		MapErrorFromRequest(w, r, err)
		return
	}
	RespondPaginated(w, data, page, perPage, total)
}

func HandleCursorPaginated(w http.ResponseWriter, r *http.Request, data interface{}, nextCursor, prevCursor *string, hasNext, hasPrev bool, limit int, err error) {
	if err != nil {
		MapErrorFromRequest(w, r, err)
		return
	}
	RespondCursorPaginated(w, data, nextCursor, prevCursor, hasNext, hasPrev, limit)
}
