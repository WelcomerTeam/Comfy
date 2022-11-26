package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	comfy "github.com/WelcomerTeam/Comfy/comfy-interactions"
	"github.com/WelcomerTeam/Discord/discord"
	protobuf "github.com/WelcomerTeam/Sandwich-Daemon/protobuf"
	sandwich "github.com/WelcomerTeam/Sandwich/sandwich"
	subway "github.com/WelcomerTeam/Subway/subway"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	PermissionsDefault = 0o744
)

func main() {
	identifierName := flag.String("identifierName", os.Getenv("IDENTIFIER_NAME"), "Sandwich identifier name")

	grpcAddress := flag.String("grpcAddress", os.Getenv("GRPC_ADDRESS"), "GRPC Address")
	proxyAddress := flag.String("proxyAddress", os.Getenv("PROXY_ADDRESS"), "Twilight proxy Address")
	prometheusAddress := flag.String("prometheusAddress", os.Getenv("PROMETHEUS_ADDRESS"), "Prometheus address")
	publicKey := flag.String("publicKey", os.Getenv("PUBLIC_KEY"), "Public key for signature validation")
	host := flag.String("host", os.Getenv("HOST"), "Host")
	webhookURL := flag.String("webhookURL", os.Getenv("WEBHOOK"), "Webhook to send status messages to")

	loggingLevel := flag.String("level", os.Getenv("LOGGING_LEVEL"), "Logging level")

	loggingFileLoggingEnabled := flag.Bool("fileLoggingEnabled", MustParseBool(os.Getenv("LOGGING_FILE_LOGGING_ENABLED")), "When enabled, will save logs to files")
	loggingEncodeAsJSON := flag.Bool("encodeAsJSON", MustParseBool(os.Getenv("LOGGING_ENCODE_AS_JSON")), "When enabled, will save logs as JSON")
	loggingCompress := flag.Bool("compress", MustParseBool(os.Getenv("LOGGING_COMPRESS")), "If true, will compress log files once reached max size")
	loggingDirectory := flag.String("directory", os.Getenv("LOGGING_DIRECTORY"), "Directory to store logs in")
	loggingFilename := flag.String("filename", os.Getenv("LOGGING_FILENAME"), "Filename to store logs as")
	loggingMaxSize := flag.Int("maxSize", MustParseInt(os.Getenv("LOGGING_MAX_SIZE")), "Maximum size for log files before being split into seperate files")
	loggingMaxBackups := flag.Int("maxBackups", MustParseInt(os.Getenv("LOGGING_MAX_BACKUPS")), "Maximum number of log files before being deleted")
	loggingMaxAge := flag.Int("maxAge", MustParseInt(os.Getenv("LOGGING_MAX_AGE")), "Maximum age in days for a log file")

	oneShot := flag.Bool("oneshot", false, "If true, will close the app after setting up the app")
	syncCommands := flag.Bool("syncCommands", false, "If true, will bulk update commands")

	proxyDebug := flag.Bool("proxyDebug", false, "Enable debug on proxy")

	flag.Parse()

	// Setup Rest
	proxyURL, err := url.Parse(*proxyAddress)
	if err != nil {
		panic(fmt.Errorf("failed to parse proxy address. url.Parse(%s): %w", *proxyAddress, err))
	}

	restInterface := discord.NewTwilightProxy(*proxyURL)
	restInterface.SetDebug(*proxyDebug)

	// Setup GRPC
	grpcConnection, err := grpc.Dial(*grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Errorf(`failed to parse grpcAddress. grpc.Dial(%s): %w`, *grpcAddress, err))
	}

	// Setup Logger
	level, err := zerolog.ParseLevel(*loggingLevel)
	if err != nil {
		panic(fmt.Errorf(`failed to parse loggingLevel. zerolog.ParseLevel(%s): %w`, *loggingLevel, err))
	}

	zerolog.SetGlobalLevel(level)

	writer := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.Stamp,
	}

	var writers []io.Writer

	writers = append(writers, writer)

	if *loggingFileLoggingEnabled {
		if err := os.MkdirAll(*loggingDirectory, PermissionsDefault); err != nil {
			log.Error().Err(err).Str("path", *loggingDirectory).Msg("Unable to create log directory")
		} else {
			lumber := &lumberjack.Logger{
				Filename:   path.Join(*loggingDirectory, *loggingFilename),
				MaxBackups: *loggingMaxBackups,
				MaxSize:    *loggingMaxSize,
				MaxAge:     *loggingMaxAge,
				Compress:   *loggingCompress,
			}

			if *loggingEncodeAsJSON {
				writers = append(writers, lumber)
			} else {
				writers = append(writers, zerolog.ConsoleWriter{
					Out:        lumber,
					TimeFormat: time.Stamp,
					NoColor:    true,
				})
			}
		}
	}

	mw := io.MultiWriter(writers...)
	logger := zerolog.New(mw).With().Timestamp().Logger()
	logger.Info().Msg("Logging configured")

	var webhook []string
	if webhookURL != nil && *webhookURL != "" {
		webhook = []string{*webhookURL}
	}

	sandwichClient := protobuf.NewSandwichClient(grpcConnection)

	context, cancel := context.WithCancel(context.Background())

	// Setup app.
	app := comfy.NewComfy(context, *identifierName, subway.SubwayOptions{
		SandwichClient:    sandwichClient,
		RESTInterface:     restInterface,
		Logger:            logger,
		PublicKey:         *publicKey,
		PrometheusAddress: *prometheusAddress,
		Webhooks:          webhook,
	})
	if err != nil {
		logger.Panic().Err(err).Msg("Exception creating app")
	}

	if *syncCommands {
		grpcInterface := sandwich.NewDefaultGRPCClient()
		configurations, err := grpcInterface.FetchConsumerConfiguration(&sandwich.GRPCContext{
			Context:        context,
			SandwichClient: sandwichClient,
		}, *identifierName)

		configuration, ok := configurations.Identifiers[*identifierName]
		if !ok {
			panic(fmt.Errorf(`failed to sync command: could not find identifier matching application "%s"`, *identifierName))
		}

		err = app.SyncCommands(context, "Bot "+configuration.Token, configuration.ID)
		if err != nil {
			panic(fmt.Errorf(`failed to sync commands. app.SyncCommands(): %w`, err))
		}
	}

	if !*oneShot {
		err = app.ListenAndServe("", *host)
		if err != nil {
			logger.Warn().Err(err).Msg("Exceptions whilst starting app")
		}
	}

	cancel()

	err = grpcConnection.Close()
	if err != nil {
		logger.Warn().Err(err).Msg("Exception whilst closing grpc client")
	}
}

func MustParseBool(str string) bool {
	boolean, _ := strconv.ParseBool(str)

	return boolean
}

func MustParseInt(str string) int {
	integer, _ := strconv.ParseInt(str, 10, 64)

	return int(integer)
}
