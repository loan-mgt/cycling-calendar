package main

import (
	"cpe/calendar/handlers"
	"cpe/calendar/logger"
	"cpe/calendar/metrics"

	"html/template"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var tpl *template.Template

func init() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		// Log error and exit if environment variables can't be loaded
		logger.Log.Warn().Err(err).Msg("Error loading .env file")
	}
	// Warn if TIMEZONE is not set
	if tz := os.Getenv("TIMEZONE"); tz == "" {
		logger.Log.Warn().Msg("TIMEZONE environment variable is not set")
	}

	// Parse templates
	tpl = template.Must(template.ParseFiles(filepath.Join("static", "index.html")))

	prometheus.Register(metrics.TotalRequests)
	prometheus.Register(metrics.ResponseStatus)
	prometheus.Register(metrics.HttpDuration)
}

func main() {
	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Shutdown goroutine
	go func() {
		<-sigChan
		logger.Log.Info().Msg("Shutting down gracefully...")
		os.Exit(0)
	}()

	r := mux.NewRouter()
	r.Use(metrics.PrometheusMiddleware)
	r.Path("/metrics").Handler(promhttp.Handler())

	// Serve dynamic index page
	r.HandleFunc("/", serveIndex).Methods("GET")

	// Serve static files like JavaScript, CSS, images, etc.
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// Serve calendar.ics route - use Tiz handler
	r.HandleFunc("/cycling-calendar.ics", handlers.GenerateTizICSHandler).Methods("GET")

	r.HandleFunc("/robots.txt", serveRobots).Methods("GET")

	// Serve sitemap.xml
	r.HandleFunc("/sitemap.xml", serveSitemap).Methods("GET")

	r.HandleFunc("/uci-classification-guide", serveGuidemap).Methods("GET")

	// check app health
	r.HandleFunc("/health", handlers.Health).Methods("GET")

	// Start HTTP server and log any errors that occur
	logger.Log.Info().Msg("Starting server on :8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		// Log any errors that occur while starting server
		logger.Log.Fatal().Err(err).Msg("Error starting server")
	}
}

// serveIndex renders the index.html Go template with environment variables
func serveIndex(w http.ResponseWriter, r *http.Request) {

	if err := tpl.Execute(w, nil); err != nil {
		// Log error if template rendering fails
		logger.Log.Error().Err(err).Msg("Error rendering template")
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

func serveRobots(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/robots.txt")
}

// serveSitemap serves the sitemap.xml file
func serveSitemap(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/sitemap.xml")
}

func serveGuidemap(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/uci-classification-guide.html")
}
