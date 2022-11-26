package plugins

import (
	"github.com/WelcomerTeam/Discord/discord"
	subway "github.com/WelcomerTeam/Subway/subway"
)

func NewPogCog() *PogCog {
	return &PogCog{
		InteractionCommands: subway.SetupInteractionCommandable(&subway.InteractionCommandable{}),
	}
}

type PogCog struct {
	InteractionCommands *subway.InteractionCommandable
}

// Assert types.

var (
	_ subway.Cog                        = (*PogCog)(nil)
	_ subway.CogWithInteractionCommands = (*PogCog)(nil)
)

// CogInfo returns information about a cog, including name and description.
func (p *PogCog) CogInfo() *subway.CogInfo {
	return &subway.CogInfo{
		Name:        "PogCog",
		Description: "This cog is poggers",
	}
}

// GetInteractionCommandable returns all interaction commands in a cog. You can optionally register commands here, however
// there is no guarantee this wont be called multiple times.
func (p *PogCog) GetInteractionCommandable() *subway.InteractionCommandable {
	return p.InteractionCommands
}

// RegisterCog is called in bot.RegisterCog, the plugin should set itself up, including commands.
func (p *PogCog) RegisterCog(b *subway.Subway) (err error) {
	// Using MustAddCommand instead of AddCommand ensures that we have set the bot up properly.
	// Any errors that occur adding a command, such as name colliosion, will result in a panic
	// when using MustX functions.
	p.InteractionCommands.MustAddInteractionCommand(&subway.InteractionCommandable{
		Name:        "pog",
		Description: "This command is very poggers",
		Handler: func(ctx *subway.InteractionContext) (*discord.InteractionResponse, error) {
			return &discord.InteractionResponse{
				Type: discord.InteractionCallbackTypeChannelMessageSource,
				Data: &discord.InteractionCallbackData{
					WebhookMessageParams: discord.WebhookMessageParams{
						Content: "<:rock:732274836038221855>📣 pog",
					},
				},
			}, nil
		},
	})

	return nil
}
