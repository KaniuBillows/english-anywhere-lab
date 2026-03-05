package http

import (
	nethttp "net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/bennyshi/english-anywhere-lab/internal/app"
	"github.com/bennyshi/english-anywhere-lab/internal/auth"
	"github.com/bennyshi/english-anywhere-lab/internal/output"
	"github.com/bennyshi/english-anywhere-lab/internal/pack"
	"github.com/bennyshi/english-anywhere-lab/internal/plan"
	"github.com/bennyshi/english-anywhere-lab/internal/progress"
	"github.com/bennyshi/english-anywhere-lab/internal/review"
	"github.com/bennyshi/english-anywhere-lab/internal/sync"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/handler"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/middleware"
)

// StaticFilesConfig holds configuration for local file serving.
// If Dir is empty, no static file route is mounted.
type StaticFilesConfig struct {
	Dir     string // local filesystem directory
	BaseURL string // URL prefix, e.g. "/static/files"
}

func NewRouter(
	application *app.App,
	authSvc *auth.Service,
	jwtMgr *auth.JWTManager,
	reviewSvc *review.Service,
	planSvc *plan.Service,
	progressSvc *progress.Service,
	packSvc *pack.Service,
	outputSvc *output.Service,
	syncSvc *sync.Service,
	staticFiles StaticFilesConfig,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(corsMiddleware)

	// Health check
	r.Get("/health", handler.Health)

	// Static file serving for local object storage
	if staticFiles.Dir != "" && staticFiles.BaseURL != "" {
		prefix := strings.TrimRight(staticFiles.BaseURL, "/")
		fileServer := nethttp.FileServer(nethttp.Dir(staticFiles.Dir))
		r.Handle(prefix+"/*", nethttp.StripPrefix(prefix, fileServer))
	}

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes
		authH := handler.NewAuthHandler(authSvc)
		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/refresh", authH.Refresh)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtMgr))

			// Profile
			profileH := handler.NewProfileHandler(authSvc)
			r.Get("/me", profileH.GetMe)
			r.Patch("/me/profile", profileH.UpdateProfile)

			// Review
			reviewH := handler.NewReviewHandler(reviewSvc)
			r.Get("/reviews/queue", reviewH.GetQueue)
			r.Post("/reviews/submit", reviewH.Submit)

			// Plan
			planH := handler.NewPlanHandler(planSvc)
			r.Post("/plans/bootstrap", planH.Bootstrap)
			r.Get("/plans/today", planH.GetToday)
			r.Post("/plans/{plan_id}/tasks/{task_id}/complete", planH.CompleteTask)

			// Progress
			progressH := handler.NewProgressHandler(progressSvc)
			r.Get("/progress/summary", progressH.GetSummary)
			r.Get("/progress/daily", progressH.GetDaily)

			// Pack
			packH := handler.NewPackHandler(packSvc)
			r.Get("/packs", packH.ListPacks)
			r.Get("/packs/{pack_id}", packH.GetDetail)
			r.Post("/packs/{pack_id}/enroll", packH.Enroll)
			r.Post("/packs/generate", packH.CreateGenerationJob)
			r.Get("/packs/generation-jobs/{job_id}", packH.GetGenerationJob)

			// Output
			outputH := handler.NewOutputHandler(outputSvc)
			r.Get("/lessons/{lesson_id}/output-tasks", outputH.ListTasks)
			r.Post("/output-tasks/{task_id}/submit", outputH.SubmitWriting)
			r.Get("/output-tasks/submissions/{submission_id}", outputH.GetSubmission)

			// Sync
			syncH := handler.NewSyncHandler(syncSvc)
			r.Post("/sync/events", syncH.PushEvents)
			r.Get("/sync/changes", syncH.PullChanges)
		})
	})

	return r
}

func corsMiddleware(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Idempotency-Key")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(nethttp.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
