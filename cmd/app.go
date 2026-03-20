package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sementrof/Weather/internal/api"
	"github.com/sementrof/Weather/internal/config"
	"github.com/sementrof/Weather/internal/deps"
	"go.uber.org/zap"
)

func findFrontendDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up a few levels and look for `frontend/index.html`.
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(wd, "frontend")
		if st, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil && !st.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	return "", fmt.Errorf("frontend/index.html not found (searched upwards from %s)", wd)
}

func main() {
	ctx := context.Background()
	cfg := config.LoadConfig()

	depends, err := deps.ProvideDependencies(ctx, cfg)

	if err != nil {
		log.Fatalf("Failed to provide dependencies: %v", err)

	}
	apiSetup := api.NewApi(depends)
	router := api.SetupRouter(apiSetup, depends.Logger)

	frontendDir, err := findFrontendDir()
	if err != nil {
		depends.Logger.Fatal("failed to find frontend dir", zap.Error(err))
	}

	// Serve frontend static files (index.html on `/`, JS/CSS under `/`).
	// Important: GET/HEAD only, so API POST endpoints aren't shadowed.
	router.PathPrefix("/").
		Methods(http.MethodGet, http.MethodHead).
		Handler(http.FileServer(http.Dir(frontendDir)))

	go func() {
		log.Println("PORT:", cfg.Port)
		if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
			depends.Logger.Error("Can't run server", zap.Error(err))
			return
		}
		depends.Logger.Info("Server is running on port", zap.String("port", cfg.Port))
	}()
	depends.Logger.Info("Server is running", zap.String("port", cfg.Port))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

}
