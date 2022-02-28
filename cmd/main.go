package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/WelcomerTeam/Comfy/comfy"
	"github.com/WelcomerTeam/Discord/discord"
	protobuf "github.com/WelcomerTeam/Sandwich-Daemon/protobuf"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	messaging "github.com/WelcomerTeam/Sandwich/messaging"
	sandwich "github.com/WelcomerTeam/Sandwich/sandwich"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	defaultStanChannelValue = "sandwich"
	defaultStanClusterValue = "cluster"

	// This has to be the same as your producer identifier.
	comfyBotIdentifier = "comfy"
)

func main() {
	grpcAddress := flag.String("grpcAddress", os.Getenv("GRPC_ADDRESS"), "GRPC Address")

	proxyAddress := flag.String("proxyAddress", os.Getenv("PROXY_ADDRESS"), "Twilight proxy Address")
	proxyDebug := flag.Bool("proxyDebug", false, "Enable debug on proxy")

	stanAddress := flag.String("stanAddress", os.Getenv("STAN_ADDRESS"), "NATs Streaming Address")
	stanCluster := flag.String("stanCluster", os.Getenv("STAN_CLUSTER"), "NATs Streaming Cluster")
	stanChannel := flag.String("stanChannel", os.Getenv("STAN_CHANNEL"), "NATs Streaming Channel")

	dryRun := flag.Bool("dryRun", false, "When enabled, bot will exit once all bots and cogs have been setup")

	flag.Parse()

	// Default flag values
	if stanChannel == nil || *stanChannel == "" {
		stanChannel = &defaultStanChannelValue
	}

	if stanCluster == nil || *stanCluster == "" {
		stanCluster = &defaultStanClusterValue
	}

	context := context.Background()

	// Setup Rest
	proxyURL, err := url.Parse(*proxyAddress)
	if err != nil {
		panic(fmt.Sprintf("url.Parse(%s): %v", *proxyAddress, err.Error()))
	}

	restInterface := discord.NewTwilightProxy(*proxyURL)
	restInterface.SetDebug(*proxyDebug)

	// Setup GRPC
	grpcConnection, err := grpc.Dial(*grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf(`grpc.Dial(%s): %v`, *grpcAddress, err.Error()))
	}

	// Setup NATs
	stanClient := messaging.NewStanMQClient()

	err = stanClient.Connect(context, "sandwich", map[string]interface{}{
		"Address": *stanAddress,
		"Cluster": *stanCluster,
		"Channel": *stanChannel,
	})
	if err != nil {
		panic(fmt.Sprintf(`stanClient.Connect(): %v`, err.Error()))
	}

	err = stanClient.Subscribe(context, *stanChannel)
	if err != nil {
		panic(fmt.Sprintf(`stanClient.Subscribe(%s): %v`, *stanChannel, err.Error()))
	}

	// Setup Logger
	writer := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.Stamp,
	}

	logger := zerolog.New(writer).With().Timestamp().Logger()

	// Setup Sandwich and Listeners
	sandwichClient := sandwich.NewSandwich(grpcConnection, restInterface, writer)

	sandwichClient.RegisterBot(comfyBotIdentifier, comfy.NewComfy(comfyBotIdentifier, sandwichClient))

	// We return if it a dry run. Any issues loading up the bot would've already caused a panic.
	if *dryRun {
		return
	}

	// Signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// Register message channels
	stanMessages := stanClient.Chan()
	grpcMessages := make(chan *protobuf.ListenResponse)

	go func() {
		for {
			grpcListener, err := sandwichClient.SandwichClient.Listen(context, &protobuf.ListenRequest{
				Identifier: "",
			})
			if err != nil {
				logger.Warn().Err(err).Msg("Failed to listen to grpc")

				time.Sleep(time.Second)
			} else {
				for {
					var lr protobuf.ListenResponse

					err = grpcListener.RecvMsg(&lr)
					if err != nil {
						logger.Warn().Err(err).Msg("Failed to receive grpc message")

						break
					} else {
						grpcMessages <- &lr
					}
				}
			}
		}
	}()

	// Event Loop
eventLoop:
	for {
		select {
		case grpcMessage := <-grpcMessages:
			var payload sandwich_structs.SandwichPayload

			err = jsoniter.Unmarshal(grpcMessage.Data, &payload)
			if err != nil {
				logger.Warn().Err(err).Msg("Failed to unmarshal grpc message")
			} else {
				err = sandwichClient.DispatchGRPCPayload(context, payload)
				if err != nil {
					logger.Warn().Err(err).Msg("Failed to dispatch grpc payload")
				}
			}
		case stanMessage := <-stanMessages:
			var payload sandwich_structs.SandwichPayload

			err = jsoniter.Unmarshal(stanMessage, &payload)
			if err != nil {
				logger.Warn().Err(err).Msg("Failed to unmarshal stan message")
			} else {
				err = sandwichClient.DispatchSandwichPayload(context, payload)
				if err != nil {
					logger.Warn().Err(err).Msg("Failed to dispatch sandwich payload")
				}
			}
		case <-signalCh:
			break eventLoop
		}
	}

	// Close sandwich
	stanClient.Unsubscribe()

	err = grpcConnection.Close()
	if err != nil {
		logger.Warn().Err(err).Msg("Exception whilst closing grpc client")
	}

	err = sandwichClient.Close()
	if err != nil {
		logger.Warn().Err(err).Msg("Exception whilst closing sandwich client")
	}
}
