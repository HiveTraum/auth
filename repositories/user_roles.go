package repositories

import (
	"auth/enums"
	"auth/functools"
	"auth/models"
	"context"
	"errors"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"strings"
)

func createUserRoleSQL() string {
	return `
		INSERT INTO user_roles(user_id, role_id) 
		VALUES ($1, $2) 
		RETURNING created, user_id, role_id;
		`
}

func getUserRolesSQL() string {
	return `
		SELECT created, user_id, role_id 
		FROM user_roles
		WHERE (array_length($1::integer[], 1) IS NULL OR user_id = ANY ($1::bigint[])) AND 
		      (array_length($2::integer[], 1) IS NULL OR role_id = ANY ($2::bigint[])) 
		LIMIT $3;
		`
}

func unwrapUserRoleScanError(err error) int {
	var e *pgconn.PgError

	if errors.As(err, &e) {
		if strings.Contains(e.Detail, "not present in table \"users\".") {
			return enums.UserNotFound
		} else if strings.Contains(e.Detail, "not present in table \"roles\".") {
			return enums.RoleNotFound
		} else if strings.Contains(e.Message, "violates unique constraint \"user_roles_pkey\"") {
			return enums.UserRoleAlreadyExist
		}
	}

	sentry.CaptureException(err)
	return enums.NotOk
}

func scanUserRole(row pgx.Row) (int, *models.UserRole) {
	ur := &models.UserRole{}

	err := row.Scan(&ur.Created, &ur.UserId, &ur.RoleId)
	if err != nil {
		sentry.CaptureException(err)
		return unwrapUserRoleScanError(err), nil
	}

	return enums.Ok, ur
}

func scanUserRoles(rows pgx.Rows, limit int) []*models.UserRole {
	userRoles := make([]*models.UserRole, limit)

	var i int32

	for rows.Next() {
		_, ur := scanUserRole(rows)
		userRoles[i] = ur
		i++
	}

	rows.Close()

	return userRoles[0:i]
}

type GetUserRoleQuery struct {
	UserId []int64
	RoleId []int64
	Limit  int
}

type getUserRoleRawQuery struct {
	Limit  int
	UserId string
	RoleId string
}

func convertGetUserRoleQueryToRaw(query GetUserRoleQuery) getUserRoleRawQuery {
	return getUserRoleRawQuery{
		Limit:  query.Limit,
		UserId: functools.Int64ListToPGArray(query.UserId),
		RoleId: functools.Int64ListToPGArray(query.RoleId),
	}
}

func CreateUserRole(db DB, ctx context.Context, userId int64, roleId int64) (int, *models.UserRole) {
	sql := createUserRoleSQL()
	row := db.QueryRow(ctx, sql, userId, roleId)
	return scanUserRole(row)
}

func GetUserRoles(db DB, ctx context.Context, query GetUserRoleQuery) []*models.UserRole {
	sql := getUserRolesSQL()
	rawQuery := convertGetUserRoleQueryToRaw(query)
	rows, err := db.Query(ctx, sql, rawQuery.UserId, rawQuery.RoleId, rawQuery.Limit)
	if err != nil {
		sentry.CaptureException(err)
		return nil
	}

	return scanUserRoles(rows, rawQuery.Limit)
}