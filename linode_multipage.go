package main

type linodeInfoPaginated struct {
	Pages   int          `json:"pages"`
	Results int          `json:"results"`
	Data    []LinodeInfo `json:"data"`
	Page    int          `json:"page"`
}

type stackScriptPaginated struct {
	Pages   int           `json:"pages"`
	Results int           `json:"results"`
	Data    []StackScript `json:"data"`
	Page    int           `json:"page"`
}

type linodeRegionPaginated struct {
	Pages   int            `json:"pages"`
	Results int            `json:"results"`
	Data    []LinodeRegion `json:"data"`
	Page    int            `json:"page"`
}

type linodeImagePaginated struct {
	Pages   int           `json:"pages"`
	Results int           `json:"results"`
	Data    []LinodeImage `json:"data"`
	Page    int           `json:"page"`
}

type linodeTypePaginated struct {
	Pages   int          `json:"pages"`
	Results int          `json:"results"`
	Data    []LinodeType `json:"data"`
	Page    int          `json:"page"`
}

// paginatedResult implementation for linodeInfoPaginated.
func (e *linodeInfoPaginated) pageNumber() int {
	return e.Page
}

func (e *linodeInfoPaginated) pageCount() int {
	return e.Pages
}

func (e *linodeInfoPaginated) data() interface{} {
	return e.Data
}

// paginatedResult implementation for stackScriptPaginated.
func (e *stackScriptPaginated) pageNumber() int {
	return e.Page
}

func (e *stackScriptPaginated) pageCount() int {
	return e.Pages
}

func (e *stackScriptPaginated) data() interface{} {
	return e.Data
}

// paginatedResult implementation for linodeRegionPaginated.
func (e *linodeRegionPaginated) pageNumber() int {
	return e.Page
}

func (e *linodeRegionPaginated) pageCount() int {
	return e.Pages
}

func (e *linodeRegionPaginated) data() interface{} {
	return e.Data
}

// paginatedResult implementation for linodeRegionPaginated.
func (e *linodeImagePaginated) pageNumber() int {
	return e.Page
}

func (e *linodeImagePaginated) pageCount() int {
	return e.Pages
}

func (e *linodeImagePaginated) data() interface{} {
	return e.Data
}

// paginatedResult implementation for linodeRegionPaginated.
func (e *linodeTypePaginated) pageNumber() int {
	return e.Page
}

func (e *linodeTypePaginated) pageCount() int {
	return e.Pages
}

func (e *linodeTypePaginated) data() interface{} {
	return e.Data
}
