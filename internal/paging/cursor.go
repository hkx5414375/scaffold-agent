// Package paging encodes bounded, resource-specific cursors for Agent responses.
package paging

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/hkx5414375/scaffold-agent/internal/projectmeta"
)

type cursor struct {
	Subject string `json:"subject"`
	Offset  int    `json:"offset"`
}

// Encode returns an opaque cursor bound to one result or plan.
func Encode(subject string, offset int) (string, error) {
	content, err := json.Marshal(cursor{Subject: subject, Offset: offset})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(content), nil
}

// Decode validates a cursor and returns its zero-based offset.
func Decode(value, subject string) (int, error) {
	if value == "" {
		return 0, nil
	}
	content, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return 0, errors.New("cursor is invalid")
	}
	var decoded cursor
	if err := projectmeta.DecodeStrict(content, &decoded); err != nil {
		return 0, errors.New("cursor is invalid")
	}
	if decoded.Subject != subject || decoded.Offset < 0 {
		return 0, errors.New("cursor does not belong to this resource")
	}
	return decoded.Offset, nil
}

// Bounds validates a requested page and returns its half-open item indexes.
func Bounds(total, offset, limit, defaultLimit, maximumLimit int) (int, int, error) {
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maximumLimit {
		return 0, 0, errors.New("page limit is outside the supported range")
	}
	if offset < 0 || offset > total {
		return 0, 0, errors.New("cursor is past the final item")
	}
	return offset, min(offset+limit, total), nil
}
