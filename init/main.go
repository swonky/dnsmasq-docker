package main

import (
	"context"
	"errors"
	"iter"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/google/dnsmasq_exporter/collector"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

const (
	prefix     = "DNSMASQ_"
	dnsmasqBin = "dnsmasq"

	exporterListenEnv = "DNSMASQ_INIT_EXPORTER_LISTEN"
	exporterAddrEnv   = "DNSMASQ_INIT_EXPORTER_DNSMASQ_ADDR"
	leasesPathEnv     = "DNSMASQ_INIT_LEASES_PATH"

	defaultExporterListen = ":9153"
	defaultDnsmasqAddr    = "localhost:53"
	defaultLeasesPath     = "/var/lib/misc/dnsmasq.leases"
)

// convertToOption converts a DNSMASQ_-prefixed environment variable name into
// a dnsmasq CLI flag. The prefix is stripped, underscores are replaced with
// hyphens, and the result is lowercased and prepended with "--".
// Example: DNSMASQ_NO_RESOLV → --no-resolv
func convertToOption(s string) string {
	return "--" + strings.ToLower(
		strings.ReplaceAll(
			strings.TrimPrefix(s, prefix),
			"_", "-",
		),
	)
}

// getEnv returns an iterator over environment variables that match the given
// prefix. Each yielded pair is (key, value), where value has been normalised:
// truthy values ("true", "yes", "on") are yielded as an empty string so they
// map to bare CLI flags; falsy values ("false", "no", "off") are skipped
// entirely.
func getEnv(prefix string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, env := range os.Environ() {
			if !strings.HasPrefix(env, prefix) {
				continue
			}
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}
			value := parts[1]
			switch strings.ToLower(value) {
			case "true", "yes", "on":
				value = ""
			case "false", "no", "off":
				continue
			}
			if !yield(parts[0], value) {
				return
			}
		}
	}
}

// formatArgument converts a key/value pair into a dnsmasq CLI argument string.
// If value is empty the flag is returned bare (e.g. "--no-resolv"); otherwise
// key and value are joined with "=" (e.g. "--listen-address=127.0.0.1").
func formatArgument(key, value string) string {
	key = convertToOption(key)
	if value == "" {
		return key
	}
	return key + "=" + value
}

// logArguments writes the resolved dnsmasq arguments to the provided logger,
// one flag per line.
func logArguments(log zerolog.Logger, args []string) {
	e := log.Info().Int("count", len(args))
	for _, arg := range args {
		e = e.Str("arg", arg)
	}
	e.Msg("starting dnsmasq")
}

// envOr returns the value of the named environment variable, or def if unset
// or empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// startExporter registers a dnsmasq Prometheus collector and starts an HTTP
// server on listenAddr exposing /metrics. The server shuts down gracefully
// when ctx is cancelled.
func startExporter(ctx context.Context, log zerolog.Logger, listenAddr, dnsmasqAddr, leasesPath string) {
	c := collector.New(collector.Config{
		DnsClient:    &dns.Client{},
		DnsmasqAddr:  dnsmasqAddr,
		LeasesPath:   leasesPath,
		ExposeLeases: false,
	})

	registry := prometheus.NewRegistry()
	registry.MustRegister(c)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	go func() {
		log.Info().Str("addr", listenAddr).Msg("exporter listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("exporter error")
		}
	}()
}

func initLogger() zerolog.Logger {
	return zerolog.New(os.Stderr).
		With().
		Timestamp().
		Str("component", "dnsmasq-init").
		Logger()
}

func main() {
	logger := initLogger()

	bin, err := exec.LookPath(dnsmasqBin)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not find dnsmasq binary")
	}

	args := make([]string, 0)
	for k, v := range getEnv(prefix) {
		args = append(args, formatArgument(k, v))
	}
	sort.Strings(args)
	logArguments(logger, args)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startExporter(
		ctx,
		logger.With().Str("component", "exporter").Logger(),
		envOr(exporterListenEnv, defaultExporterListen),
		envOr(exporterAddrEnv, defaultDnsmasqAddr),
		envOr(leasesPathEnv, defaultLeasesPath),
	)

	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		logger.Fatal().Err(err).Msg("failed to start dnsmasq")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		s := <-sig
		logger.Info().Str("signal", s.String()).Msg("forwarding signal to dnsmasq")
		cmd.Process.Signal(s)
	}()

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			logger.Error().Int("code", exitErr.ExitCode()).Msg("dnsmasq exited with error")
			os.Exit(exitErr.ExitCode())
		}
		logger.Fatal().Err(err).Msg("dnsmasq exited unexpectedly")
	}
}
