package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

const (
	DESTINATION_STDOUT = "STDOUT"
	DESTINATION_STDERR = "STDERR"
)

type options struct {
	port          uint16
	netInterface  net.IP
	body          string
	bodyDelimiter string
	headers       string
	log           string
	logLevel      int
}

var logLevelMap = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}
var logLevels = []slog.Level{
	slog.LevelError,
	slog.LevelWarn,
	slog.LevelInfo,
	slog.LevelDebug,
}

func initFlags() (options, error) {
	portFlag := flag.Uint("port", 8080, "port to bind to")
	flag.UintVar(portFlag, "p", *portFlag, "alias for -port")

	interfaceFlag := flag.String("interface", "127.0.0.1", "network interface to bind to")
	flag.StringVar(interfaceFlag, "i", *interfaceFlag, "alias for -interface")

	bodyFlag := flag.String("body", DESTINATION_STDOUT, "where to write the request body. Valid options are STDOUT, STDERR or a path to a file.")
	flag.StringVar(bodyFlag, "b", *bodyFlag, "alias for -body")

	bodyDelimFlag := flag.String("bdelim", "\n", "what to write after writing the request body.")

	headersFlag := flag.String("headers", "", "where to write the request headers. Valid options are STDOUT, STDERR or a path to a file.")
	flag.StringVar(headersFlag, "H", *headersFlag, "alias for -headers")

	logFlag := flag.String("log", DESTINATION_STDERR, "where to write logs. Logs will be discarded if you set any output flag to STDERR. Valid options are STDOUT, STDERR or a path to a file.")
	flag.StringVar(logFlag, "l", *logFlag, "alias for -log")

	logLevelFlag := flag.String("verbosity", "", "logging verbosity. Valid options are error, warn, info, debug.")

	flag.Parse()

	netInterface := net.ParseIP(*interfaceFlag)
	if netInterface == nil {
		return options{}, fmt.Errorf("could not parse interface flag as IP interface=%v", *interfaceFlag)
	}

	if *portFlag > math.MaxUint16 {
		return options{}, fmt.Errorf("port flag invalid (greater than %v)", math.MaxUint16)
	}
	port := uint16(*portFlag)

	body := *bodyFlag
	bodyDelimiter := *bodyDelimFlag
	headers := *headersFlag
	log := *logFlag
	var logLevel int = int(slog.LevelError) + 1
	if *logLevelFlag != "" {
		if val, ok := logLevelMap[*logLevelFlag]; ok {
			logLevel = int(val)
		}
	}

	return options{
		port,
		netInterface,
		body,
		bodyDelimiter,
		headers,
		log,
		logLevel,
	}, nil
}

func initLogging(opts options) (*slog.Logger, error) {
	var writer io.Writer
	if opts.log == DESTINATION_STDOUT {
		if opts.body == DESTINATION_STDOUT || opts.headers == DESTINATION_STDOUT {
			writer = io.Discard
		} else {
			writer = os.Stdout
		}
	} else if opts.log == DESTINATION_STDERR {
		if opts.body == DESTINATION_STDERR || opts.headers == DESTINATION_STDERR {
			writer = io.Discard
		} else {
			writer = os.Stderr
		}
	} else {
		logFile, err := os.Open(opts.log)
		if err != nil {
			return nil, fmt.Errorf("could not open log file at %v: %w", opts.log, err)
		}

		writer = logFile
	}

	var logLevel = new(slog.LevelVar)
	logLevel.Set(slog.Level(opts.logLevel))
	logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: logLevel}))
	return logger, nil
}

func main() {
	log.SetOutput(os.Stderr)
	opts, err := initFlags()
	if err != nil {
		log.Fatalf("Could not parse flags: %v", err)
	}

	logger, err := initLogging(opts)
	if err != nil {
		logger.Error("Could not initialize logging", slog.Any("error", err))
		os.Exit(1)
	}

	var bodyDest io.Writer
	if opts.body == DESTINATION_STDOUT {
		bodyDest = os.Stdout
	} else if opts.body == DESTINATION_STDERR {
		bodyDest = os.Stderr
	} else {
		bodyDest, err = os.Create(opts.body)
		if err != nil {
			logger.Error("Could not open file for writing the request bodies",
				slog.String("file", opts.body),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
	}

	logger.Info("Initialization finished",
		slog.String("body destination", opts.body),
		slog.String("log destination", opts.log),
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			return
		}
		rlogger := logger.With(
			slog.Any("datetime", time.Now()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.String()),
		)

		bodyReader := io.TeeReader(r.Body, bodyDest)
		if _, err := io.ReadAll(bodyReader); err != nil {
			rlogger.Error("Could not read request body for request", slog.Any("error", err))
		}
		if _, err = bodyDest.Write([]byte(opts.bodyDelimiter)); err != nil {
			rlogger.Error("Could not write delimiter after writing body", slog.Any("error", err))
		}
	})

	server := http.Server{
		Addr:    fmt.Sprintf("%v:%v", opts.netInterface, opts.port),
		Handler: handler,
	}

	go func() {
		if err = server.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("HTTP server crashed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)

	<-interruptChan
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err = server.Shutdown(ctx); err != nil {
		logger.Error("Could not shutdown server gracefully")
		cancel()
		os.Exit(1)
	}
	cancel()

	os.Exit(0)
}
