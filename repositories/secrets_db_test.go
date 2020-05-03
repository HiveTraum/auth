package repositories

import (
	"auth/config"
	"context"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateSecret(t *testing.T) {
	pool := config.InitPool(nil, config.InitEnvironment())
	ctx := context.Background()
	PurgeSecrets(pool, ctx)
	secret := CreateSecret(pool, ctx)
	require.NotNil(t, secret)
	require.NotNil(t, secret.Value)
}

func TestGetSecretFromDB(t *testing.T) {
	pool := config.InitPool(nil, config.InitEnvironment())
	ctx := context.Background()
	PurgeSecrets(pool, ctx)
	createdSecret := CreateSecret(pool, ctx)
	secret := GetSecretFromDB(pool, ctx, createdSecret.Id)
	require.NotNil(t, secret)
	require.Equal(t, createdSecret, secret)
}

func TestGetSecretFromDBWithoutSecret(t *testing.T) {
	pool := config.InitPool(nil, config.InitEnvironment())
	ctx := context.Background()
	PurgeSecrets(pool, ctx)
	secret := GetSecretFromDB(pool, ctx, uuid.NewV4())
	require.Nil(t, secret)
}