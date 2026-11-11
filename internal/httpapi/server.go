package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/state"
)

// bookSource — всё, что серверу нужно от владельца.
type bookSource interface {
	Book(ctx context.Context, name string) (state.BookView, error)
}

type lotJSON struct {
	ID        int64     `json:"id"`
	Price     string    `json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
	Age       string    `json:"age"` // сколько времени назад лот обновлялся
}

type bookJSON struct {
	Name string    `json:"name"`
	Lots []lotJSON `json:"lots"`
}

// NewHandler возвращает обработчик с маршрутом GET /book?name=...
func NewHandler(src bookSource) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /book", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "query parameter 'name' is required", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		v, err := src.Book(ctx, name)
		if err != nil {
			http.Error(w, "owner is busy: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		if !v.Found {
			http.Error(w, "no book for "+name, http.StatusNotFound)
			return
		}

		resp := bookJSON{Name: v.Name, Lots: make([]lotJSON, 0, len(v.Lots))}
		for _, l := range v.Lots {
			resp.Lots = append(resp.Lots, lotJSON{
				ID:        l.ID,
				Price:     l.Price.String(),
				UpdatedAt: l.UpdatedAt,
				Age:       time.Since(l.UpdatedAt).Round(time.Second).String(),
			})
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(resp)
	})

	return mux
}

// Run запускает сервер и останавливает его, когда отменён ctx.
func Run(ctx context.Context, addr string, h http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
