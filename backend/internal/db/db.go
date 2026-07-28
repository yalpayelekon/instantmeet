package db

import (
	"context"
	"fmt"
	"time"

	"github.com/instantmeet/instantmeet/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string) (*Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Pool{pool: pool}, nil
}

func (p *Pool) Close() {
	if p != nil && p.pool != nil {
		p.pool.Close()
	}
}

func (p *Pool) UpsertUser(ctx context.Context, user models.User) error {
	const q = `
INSERT INTO users (id, google_id, email, display_name, avatar_url, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
  google_id = EXCLUDED.google_id,
  email = EXCLUDED.email,
  display_name = EXCLUDED.display_name,
  avatar_url = EXCLUDED.avatar_url,
  updated_at = NOW()`
	googleID := user.GoogleID
	if googleID == "" {
		googleID = user.ID
	}
	_, err := p.pool.Exec(ctx, q, user.ID, googleID, user.Email, user.DisplayName, user.Avatar)
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}
	return nil
}

func (p *Pool) GetByID(ctx context.Context, id string) (models.User, error) {
	const q = `SELECT id, google_id, email, display_name, avatar_url FROM users WHERE id = $1`
	var user models.User
	err := p.pool.QueryRow(ctx, q, id).Scan(&user.ID, &user.GoogleID, &user.Email, &user.DisplayName, &user.Avatar)
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}
