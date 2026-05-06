package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://testuser:testpass@localhost:5433/controlplane_test?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	fmt.Println("Applying migrations...")
	migrations := []string{
		"migrations/0001_initial_schema.sql",
		"migrations/0002_create_audit_logs.sql",
		"migrations/0003_create_sessions_table.sql",
	}
	for _, file := range migrations {
		sql, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read %s: %v", file, err)
		}
		_, err = pool.Exec(ctx, string(sql))
		if err != nil {
			log.Printf("Migration warning (might already exist): %v", err)
		}
	}

	fmt.Println("Seeding database...")

	// 1. Create Teams
	teamMap := make(map[string]int)
	teams := []string{"Engineering", "Operations"}

	for _, teamName := range teams {
		var teamID int
		err = pool.QueryRow(ctx, `
			INSERT INTO teams (name) 
			VALUES ($1) 
			ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name 
			RETURNING id`, teamName).Scan(&teamID)
		if err != nil {
			log.Fatalf("Failed to insert team %s: %v", teamName, err)
		}
		teamMap[teamName] = teamID
	}
	fmt.Println("✓ Teams created")

	// 2. Create Users with Demo Credentials
	users := []struct {
		Email    string
		Password string
		Role     string
		Team     string
	}{
		{"alice@example.com", "password123", "admin", "Engineering"},
		{"bob@example.com", "password123", "reviewer", "Engineering"},
		{"charlie@example.com", "password123", "member", "Engineering"},
		{"diana@example.com", "password123", "member", "Operations"},
	}

	for _, u := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), 12)
		if err != nil {
			log.Fatal(err)
		}

		var id int
		err = pool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash) 
			VALUES ($1, $2) 
			ON CONFLICT (email) DO UPDATE SET email=EXCLUDED.email 
			RETURNING id`, u.Email, hash).Scan(&id)
		if err != nil {
			log.Fatalf("Failed to insert user %s: %v", u.Email, err)
		}

		teamID := teamMap[u.Team]
		_, err = pool.Exec(ctx, `
			INSERT INTO memberships (user_id, team_id, role) 
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING`, id, teamID, u.Role)
		if err != nil {
			log.Fatalf("Failed to insert membership for %s: %v", u.Email, err)
		}
		fmt.Printf("✓ Created user: %s (role: %s, team: %s)\n", u.Email, u.Role, u.Team)
	}

	// 2. Create a Notebook
	var notebookID int
	err = pool.QueryRow(ctx, `
		INSERT INTO notebooks (title, content, slug, visibility, is_published, team_id)
		VALUES ('Incident Response Plan', 'This is the incident response plan.', 'incident-response-plan', 'public', false, 1)
		ON CONFLICT DO NOTHING
		RETURNING id`).Scan(&notebookID)

	if err != nil && err.Error() != "no rows in result set" {
		log.Fatalf("Failed to insert notebook: %v", err)
	}

	if notebookID != 0 {
		// Create a revision for this notebook
		_, err = pool.Exec(ctx, `
			INSERT INTO notebook_revisions (notebook_id, author_id, title, body, status)
			VALUES ($1, 1, 'Incident Response Plan', 'This is the first draft of the incident response plan.', 'draft')`, notebookID)
		if err != nil {
			log.Fatalf("Failed to insert revision: %v", err)
		}
	}

	fmt.Println("Seed completed successfully!")
}
