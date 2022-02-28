package comfy

import (
	"fmt"
	"github.com/WelcomerTeam/Comfy/comfy/plugins"
	"github.com/WelcomerTeam/Discord/discord"
	sandwich "github.com/WelcomerTeam/Sandwich/sandwich"
	"golang.org/x/xerrors"
)

func NewComfy(identifierName string, sandwichClient *sandwich.Sandwich) (bot *sandwich.Bot) {
	bot = sandwich.NewBot(sandwich.WhenMentionedOr("+"), sandwichClient.Logger)

	bot.MustRegisterCog(plugins.NewGeneralCog())
	bot.MustRegisterCog(plugins.NewPogCog())

	// Register application commands

	// context := context.Background()
	// session := discord.NewSession(context, "", sandwichClient.RESTInterface, sandwichClient.Logger)

	// identifier, ok, err := sandwichClient.FetchIdentifier(context, identifierName)
	// if err != nil {
	// 	sandwichClient.Logger.Warn().Err(err).Str("identifier", identifierName).Msg("Failed to fetch identifier")
	// }

	// if ok {
	// 	session.Token = "Bot " + identifier.Token

	// 	applicationCommands := bot.InteractionCommands.MapApplicationCommands()

	// 	_, err = discord.BulkOverwriteGlobalApplicationCommands(session, identifier.ID, applicationCommands)
	// 	if err != nil {
	// 		sandwichClient.Logger.Warn().Err(err).Msg("Failed to override global application commands")
	// 	}
	// }

	bot.RegisterOnInteractionCreateEvent(func(ctx *sandwich.EventContext, interaction discord.Interaction) (err error) {
		resp, err := bot.ProcessInteraction(ctx, interaction)
		if err != nil {
			fmt.Println(err.Error())

			return
		}

		if resp != nil {
			err = interaction.SendResponse(ctx.Session, resp.Type, resp.Data.WebhookMessageParams, resp.Data.Choices)
			if err != nil {
				fmt.Println(err.Error())

				return
			}
		}

		return nil
	})

	bot.RegisterOnMessageCreateEvent(func(ctx *sandwich.EventContext, message discord.Message) (err error) {
		err = bot.ProcessCommands(ctx, message)
		if err != nil {
			ctx.Logger.Warn().Err(err).Str("content", message.Content).Msg("Failed to process command")

			return xerrors.Errorf("Failed to process command: %v", err)
		}

		return nil
	})

	return bot
}
