package valueobject

type TodoFilters struct {
	Query   string
	Page    int
	PerPage int
}

func NewTodoFilters(query string, page, perPage int) TodoFilters {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return TodoFilters{
		Query:   query,
		Page:    page,
		PerPage: perPage,
	}
}

func (f TodoFilters) Offset() int {
	return (f.Page - 1) * f.PerPage
}
