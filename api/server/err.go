package server

import (
	"errors"
	"strings"

	"github.com/mattn/go-sqlite3"
)

const (
	ErrMissingArgEmail     = "missing argument: email"
	ErrMissingArgID        = "missing argument: id"
	ErrMissingArgUserID    = "missing argument: user id"
	ErrMissingArgName      = "missing argument: name"
	ErrMissingArgUsername  = "missing argument: username"
	ErrMissingArgPassword  = "missing argument: password"
	ErrMissingArgDomain    = "missing argument: domain"
	ErrMissingArgProtocol  = "missing argument: protocol"
	ErrMissingArgTarget    = "missing argument: target"
	ErrMissingArgIPV4      = "missing argument: ipv4"
	ErrMissingRefreshToken = "missing refresh token"
	ErrMissingMetadata     = "missing metadata"
	ErrMissingCookieHeader = "missing cookie header"
	ErrMissingCtxPayload   = "missing ctx payload"

	ErrFailedToCreateTunnel         = "failed to create tunnel"
	ErrFailedtoGetTunnel            = "failed to get tunnel"
	ErrFailedToUpdateTunnel         = "failed to update tunnel"
	ErrFailedToDeleteTunnel         = "failed to delete tunnel"
	ErrFailedToCreateUser           = "failed to create user"
	ErrFailedToGetUser              = "failed to get user"
	ErrFailedToUpdateUser           = "failed to update user"
	ErrFailedToDeleteUser           = "failed to delete user"
	ErrFailedToGenerateHash         = "failed to generate hash"
	ErrFailedToGenerateAccessToken  = "failed to generate access token"
	ErrFailedToGenerateRefreshToken = "failed to generate refresh token"
	ErrFailedToRefreshToken         = "failed to refresh token"
	ErrFailedToGetRefreshToken      = "failed to get refresh tokenn"
	ErrFailedToGetTunnel            = "failed to get tunnel"

	ErrUsernameAlreadyUsed     = "username is already used"
	ErrEmailAlreadyUsed        = "email is already used"
	ErrTunnelNameAlreadyUsed   = "name is already used"
	ErrTunnelDomainAlreadyUsed = "domain is already used"

	ErrRefreshTokenNotFound = "refresh token not found"
	ErrUserNotFound         = "user not found"
	ErrTunnelNotFound       = "tunnel not found"
	ErrInvalidRefreshToken  = "invalid refresh token"
	ErrInvalidPassword      = "invalid password"
	ErrInternal             = "internal error"
)

func isUniqueConstraint(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
}

func isUniqueConstraintOn(err error, column string) bool {
	return isUniqueConstraint(err) && strings.Contains(err.Error(), column)
}
