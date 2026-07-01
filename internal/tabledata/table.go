package tabledata

import "time"

type Header struct {
	Title string
	Key   string
	Align int
}

type Row struct {
	ID        string
	Fields    []string
	SortKey   map[string]string
	UpdatedAt time.Time
	ColorKey  string
	Raw       any
}
