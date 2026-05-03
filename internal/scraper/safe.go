package scraper

import (
	"context"
	"fmt"

	"github.com/henry/javapi/internal/domain"
)

// SafeSearch calls s.Search(ctx, code) and recovers from any panic, returning
// the panic as an error with the scraper name included.
func SafeSearch(ctx context.Context, s domain.Scraper, code string) (results []domain.VideoResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("scraper %s panicked: %v", s.Name(), r)
		}
	}()
	return s.Search(ctx, code)
}
