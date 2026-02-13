package config

import (
	"testing"
)

func TestDetectDatabaseDriver(t *testing.T) {
	tests := []struct {
		name       string
		yamlDriver string
		dbURL      string
		want       string
	}{
		{"YAML sqlite", "sqlite", "", "sqlite"},
		{"YAML postgres", "postgres", "", "postgres"},
		{"YAML SQLITE uppercase", "SQLite", "", "sqlite"},
		{"YAML Postgres mixed", "Postgres", "", "postgres"},
		{"URL file: prefix", "", "file:/var/lib/test.db?cache=shared", "sqlite"},
		{"URL sqlite: prefix", "", "sqlite:///tmp/test.db", "sqlite"},
		{"URL postgres:// prefix", "", "postgres://user:pass@localhost:5432/db", "postgres"},
		{"URL postgresql:// prefix", "", "postgresql://user:pass@localhost:5432/db", "postgres"},
		{"YAML overrides URL", "sqlite", "postgres://user:pass@localhost:5432/db", "sqlite"},
		{"empty defaults to mongodb", "", "", "mongodb"},
		{"unknown defaults to mongodb", "", "mysql://localhost/db", "mongodb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectDatabaseDriver(tt.yamlDriver, tt.dbURL)
			if got != tt.want {
				t.Errorf("detectDatabaseDriver(%q, %q) = %q, want %q", tt.yamlDriver, tt.dbURL, got, tt.want)
			}
		})
	}
}

func TestBuildDatabaseURL(t *testing.T) {
	tests := []struct {
		name     string
		db       DatabaseConfig
		password string
		wantPfx  string // expected URL prefix
		wantSub  string // expected substring
	}{
		{
			name:     "postgres default",
			db:       DatabaseConfig{Driver: "postgres", Host: "db.local", Port: 5432, User: "admin", Name: "mydb", SSLMode: "disable"},
			password: "secret",
			wantPfx:  "postgres://",
			wantSub:  "db.local:5432/mydb",
		},
		{
			name:     "postgres empty driver (backward compat)",
			db:       DatabaseConfig{Host: "db.local", Port: 5432, User: "admin", Name: "mydb", SSLMode: "disable"},
			password: "secret",
			wantPfx:  "postgres://",
			wantSub:  "db.local:5432/mydb",
		},
		{
			name:    "sqlite with path",
			db:      DatabaseConfig{Driver: "sqlite", Path: "/data/test.db"},
			wantPfx: "file:",
			wantSub: "/data/test.db?cache=shared",
		},
		{
			name:    "sqlite default path",
			db:      DatabaseConfig{Driver: "sqlite"},
			wantPfx: "file:",
			wantSub: "/var/lib/agents-admin/agents-admin.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildDatabaseURL(tt.db, tt.password)
			if len(got) < len(tt.wantPfx) || got[:len(tt.wantPfx)] != tt.wantPfx {
				t.Errorf("buildDatabaseURL() = %q, want prefix %q", got, tt.wantPfx)
			}
			if tt.wantSub != "" {
				found := false
				for i := 0; i <= len(got)-len(tt.wantSub); i++ {
					if got[i:i+len(tt.wantSub)] == tt.wantSub {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("buildDatabaseURL() = %q, want substring %q", got, tt.wantSub)
				}
			}
		})
	}
}

func TestBuildRedisURL(t *testing.T) {
	tests := []struct {
		name string
		cfg  RedisConfig
		want string
	}{
		{
			name: "no password",
			cfg:  RedisConfig{Host: "localhost", Port: 6380, DB: 0},
			want: "redis://localhost:6380/0",
		},
		{
			name: "with password",
			cfg:  RedisConfig{Host: "localhost", Port: 6380, DB: 0, Password: "secret"},
			want: "redis://:secret@localhost:6380/0",
		},
		{
			name: "URL takes precedence",
			cfg:  RedisConfig{Host: "localhost", Port: 6380, DB: 0, Password: "secret", URL: "redis://other:6379/1"},
			want: "redis://other:6379/1",
		},
		{
			name: "with password and db",
			cfg:  RedisConfig{Host: "redis.local", Port: 6379, DB: 2, Password: "p@ss"},
			want: "redis://:p@ss@redis.local:6379/2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRedisURL(tt.cfg)
			if got != tt.want {
				t.Errorf("buildRedisURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildDatabaseURL_MongoDB(t *testing.T) {
	tests := []struct {
		name     string
		db       DatabaseConfig
		password string
		want     string
	}{
		{
			name: "mongodb no auth",
			db:   DatabaseConfig{Driver: "mongodb", Host: "localhost", Port: 27017},
			want: "mongodb://localhost:27017",
		},
		{
			name:     "mongodb with auth",
			db:       DatabaseConfig{Driver: "mongodb", Host: "localhost", Port: 27017, User: "admin"},
			password: "secret",
			want:     "mongodb://admin:secret@localhost:27017",
		},
		{
			name: "mongodb URI takes precedence",
			db:   DatabaseConfig{Driver: "mongodb", Host: "localhost", Port: 27017, User: "admin", URI: "mongodb://custom:27017"},
			want: "mongodb://custom:27017",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildDatabaseURL(tt.db, tt.password)
			if got != tt.want {
				t.Errorf("buildDatabaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"postgres://user:secret@localhost:5432/db", "postgres://user:***@localhost:5432/db"},
		{"file:/var/lib/test.db", "file:/var/lib/test.db"},
		{"redis://localhost:6379/0", "redis://localhost:6379/0"},
	}
	for _, tt := range tests {
		got := maskPassword(tt.input)
		if got != tt.want {
			t.Errorf("maskPassword(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseEnv(t *testing.T) {
	tests := []struct {
		input string
		want  Environment
	}{
		{"dev", EnvDevelopment},
		{"test", EnvTest},
		{"prod", EnvProduction},
		{"production", EnvProduction},
		{"", EnvDevelopment},
		{"unknown", EnvDevelopment},
	}
	for _, tt := range tests {
		got := parseEnv(tt.input)
		if got != tt.want {
			t.Errorf("parseEnv(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestConfigString(t *testing.T) {
	cfg := &Config{
		Env:            EnvProduction,
		DatabaseDriver: "sqlite",
		DatabaseURL:    "file:/var/lib/agents-admin/agents-admin.db?cache=shared&mode=rwc",
		RedisURL:       "redis://localhost:6379/0",
	}
	s := cfg.String()
	if s == "" {
		t.Error("Config.String() should not be empty")
	}
	// Should contain driver
	for _, want := range []string{"sqlite", "prod"} {
		found := false
		for i := 0; i <= len(s)-len(want); i++ {
			if s[i:i+len(want)] == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Config.String() = %q, should contain %q", s, want)
		}
	}
}
