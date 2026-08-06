package contract

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/app/query"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestBuiltInTemplatesLoad(t *testing.T) {
	db, err := storesqlite.Open(context.Background(), "file:templates?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := storesqlite.NewTemplateRepository(db)
	service := query.NewTemplateService(repo)
	if err = service.LoadBuiltins(context.Background(), filepath.Join("..", "..", "templates")); err != nil {
		t.Fatal(err)
	}
	values, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 9 {
		t.Fatalf("templates=%d", len(values))
	}
	for _, template := range values {
		if template.Key == "nginx-container" {
			if len(template.Versions) != 2 {
				t.Fatalf("nginx versions=%d", len(template.Versions))
			}
			return
		}
	}
	t.Fatal("nginx-container template missing")
}
