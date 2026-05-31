package apitype

import (
	"encoding/json"
	"time"
)

type Date struct {
	time.Time
}

func NewDate(t time.Time) Date {
	return Date{Time: t}
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format("2006-01-02"))
}

func (d *Date) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

type LocalDateTime struct {
	time.Time
}

func NewLocalDateTime(t time.Time) LocalDateTime {
	return LocalDateTime{Time: t}
}

func (d LocalDateTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format("2006-01-02T15:04:05"))
}

func (d *LocalDateTime) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	layouts := []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05", time.RFC3339}
	var parseErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			d.Time = parsed
			return nil
		}
		parseErr = err
	}
	return parseErr
}
