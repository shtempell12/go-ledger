package db

import "testing"

func TestDSN(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"defaults", nil, "postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"},
		{
			"separate variables",
			map[string]string{"DB_HOST": "db", "DB_PORT": "6000", "DB_USER": "u", "DB_PASS": "p@ss/1", "DB_NAME": "cash"},
			"postgres://u:p%40ss%2F1@db:6000/cash?sslmode=disable",
		},
		{
			"DATABASE_URL wins",
			map[string]string{"DATABASE_URL": "postgres://a:b@c:1/d", "DB_HOST": "db"},
			"postgres://a:b@c:1/d",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, key := range []string{"DATABASE_URL", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASS", "DB_NAME"} {
				t.Setenv(key, c.env[key])
			}
			if got := DSN(); got != c.want {
				t.Fatalf("want %s, got %s", c.want, got)
			}
		})
	}
}
