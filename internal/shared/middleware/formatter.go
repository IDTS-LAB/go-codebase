package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
)

func ResponseFormatter() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fw := &formattingWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(fw, r)

			if len(fw.body) == 0 {
				w.WriteHeader(fw.statusCode)
				return
			}

			if isEnvelope(fw.body) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(fw.statusCode)
				w.Write(fw.body)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(fw.statusCode)

			if fw.statusCode >= 400 {
				var errBody struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				}
				if json.Unmarshal(fw.body, &errBody) == nil && errBody.Message != "" {
					json.NewEncoder(w).Encode(utils.APIResponse{
						Success: false,
						Error:   &utils.ErrorBody{Code: errBody.Code, Message: errBody.Message},
					})
					return
				}
				json.NewEncoder(w).Encode(utils.APIResponse{
					Success: false,
					Error:   &utils.ErrorBody{Code: http.StatusText(fw.statusCode), Message: string(bytes.TrimSpace(fw.body))},
				})
				return
			}

			var paginated struct {
				Data       interface{} `json:"data"`
				Pagination interface{} `json:"pagination"`
			}
			if json.Unmarshal(fw.body, &paginated) == nil && paginated.Data != nil && paginated.Pagination != nil {
				var meta utils.PaginationMeta
				metaBytes, _ := json.Marshal(paginated.Pagination)
				json.Unmarshal(metaBytes, &meta)
				json.NewEncoder(w).Encode(utils.APIResponse{
					Success: true,
					Data:    paginated.Data,
					Meta:    &meta,
				})
				return
			}

			var raw interface{}
			json.Unmarshal(fw.body, &raw)
			json.NewEncoder(w).Encode(utils.APIResponse{
				Success: true,
				Data:    raw,
				Meta:    nil,
			})
		})
	}
}

type formattingWriter struct {
	http.ResponseWriter
	statusCode  int
	body        []byte
	wroteHeader bool
}

func (w *formattingWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.statusCode = code
	w.wroteHeader = true
}

func (w *formattingWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}

func (w *formattingWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func isEnvelope(body []byte) bool {
	var envelope struct {
		Success *bool `json:"success"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false
	}
	return envelope.Success != nil
}
