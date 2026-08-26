package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/doclens/identity-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultOperationTimeout = 5 * time.Second

// Postgres is the durable identity store. Every write is committed in a
// transaction and every operation has a bounded context.
type Postgres struct {
	pool             *pgxpool.Pool
	operationTimeout time.Duration
}

func NewPostgres(ctx context.Context, databaseURL string, operationTimeout time.Duration) (*Postgres, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL is required")
	}
	if operationTimeout <= 0 {
		operationTimeout = defaultOperationTimeout
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	store := &Postgres{pool: pool, operationTimeout: operationTimeout}
	pingCtx, cancel := store.operationContext(ctx)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (p *Postgres) Close() {
	p.pool.Close()
}

func (p *Postgres) Migrate(ctx context.Context) error {
	opCtx, cancel := p.operationContext(ctx)
	defer cancel()
	tx, err := p.pool.BeginTx(opCtx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(opCtx, initialMigrationSQL); err != nil {
		return err
	}
	return tx.Commit(opCtx)
}

func (p *Postgres) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	opCtx, cancel := p.operationContext(ctx)
	defer cancel()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	roles := user.Roles
	if roles == nil {
		roles = []string{}
	}
	tx, err := p.pool.BeginTx(opCtx, pgx.TxOptions{})
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(context.Background()) // safe after commit

	_, err = tx.Exec(opCtx, `
		INSERT INTO users (id, organization_id, email, password_hash, roles, disabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, user.OrganizationID, normalizeEmail(user.Email), user.PasswordHash, roles, user.Disabled, user.CreatedAt.UTC())
	if isUniqueViolation(err) {
		return domain.User{}, ErrUserExists
	}
	if err != nil {
		return domain.User{}, err
	}
	if err := tx.Commit(opCtx); err != nil {
		return domain.User{}, err
	}
	user.Email = normalizeEmail(user.Email)
	return user, nil
}

func (p *Postgres) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	opCtx, cancel := p.operationContext(ctx)
	defer cancel()
	var user domain.User
	err := p.pool.QueryRow(opCtx, `
		SELECT id, organization_id, email, password_hash, roles, disabled, created_at
		FROM users
		WHERE email = $1
	`, normalizeEmail(email)).Scan(
		&user.ID,
		&user.OrganizationID,
		&user.Email,
		&user.PasswordHash,
		&user.Roles,
		&user.Disabled,
		&user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (p *Postgres) SaveRefreshToken(ctx context.Context, token domain.RefreshToken) error {
	opCtx, cancel := p.operationContext(ctx)
	defer cancel()
	tx, err := p.pool.BeginTx(opCtx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(opCtx, `
		INSERT INTO refresh_tokens (token_hash, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
	`, hashToken(token.Token), token.UserID, token.ExpiresAt.UTC(), token.CreatedAt.UTC())
	if err != nil {
		return err
	}
	return tx.Commit(opCtx)
}

func (p *Postgres) operationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, p.operationTimeout)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
