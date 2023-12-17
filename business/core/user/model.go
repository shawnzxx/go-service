package user

import (
	"net/mail"
	"time"

	"github.com/google/uuid"
)

// User represents information about an individual user.
// here each filed we try to use type instead of string as much as possible
// so that upper layer need to use parse to covert to the type filed which help to do the validation.
type User struct {
	ID           uuid.UUID
	Name         string
	Email        mail.Address
	Roles        []Role
	PasswordHash []byte
	Department   string
	Enabled      bool
	DateCreated  time.Time
	DateUpdated  time.Time
}

// NewUser contains information needed to create a new user.
// for CRUD we need to create separate struct for each operation instead of re-use same struct
type NewUser struct {
	Name            string
	Email           mail.Address
	Roles           []Role
	Department      string
	Password        string
	PasswordConfirm string
}

// UpdateUser contains information needed to update a user.
// we use pointer semantic let upper layer decide what are fields needs to update
// instead of provide the whole struct, because of pointer so if not provide will be nil
type UpdateUser struct {
	Name            *string
	Email           *mail.Address
	Roles           []Role
	Department      *string
	Password        *string
	PasswordConfirm *string
	Enabled         *bool
}
