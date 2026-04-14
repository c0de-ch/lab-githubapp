package sse

import "fmt"

type Event struct {
	Name string
	Data string
	ID   string
}

func (e Event) Format() []byte {
	var buf []byte
	if e.Name != "" {
		buf = append(buf, fmt.Sprintf("event: %s\n", e.Name)...)
	}
	if e.ID != "" {
		buf = append(buf, fmt.Sprintf("id: %s\n", e.ID)...)
	}
	buf = append(buf, fmt.Sprintf("data: %s\n", e.Data)...)
	buf = append(buf, '\n')
	return buf
}
