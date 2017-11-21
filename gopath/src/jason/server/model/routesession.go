package model

import (
	"github.com/satori/go.uuid"
)

func newToken() string {
	return uuid.NewV4().String()
}
