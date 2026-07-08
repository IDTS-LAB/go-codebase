package pagination

import (
	"math"
	"net/http"
	"strconv"
)

type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

func NewPagination(r *http.Request, defaultPerPage, maxPerPage int) (page, perPage int) {
	page = 1
	perPage = defaultPerPage

	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}

	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if val, err := strconv.Atoi(pp); err == nil && val > 0 {
			perPage = val
		}
	}

	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	return page, perPage
}

func (p *Pagination) Calculate(total int) {
	p.Total = total
	p.TotalPages = int(math.Ceil(float64(total) / float64(p.PerPage)))
}

func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PerPage
}

func NewPaginatedResponse(data interface{}, page, perPage, total int) PaginatedResponse {
	p := Pagination{
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}
	p.Calculate(total)
	return PaginatedResponse{
		Data:       data,
		Pagination: p,
	}
}
