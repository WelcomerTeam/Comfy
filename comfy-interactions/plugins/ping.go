package plugins

import (
	"fmt"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	subway "github.com/WelcomerTeam/Subway/subway"
)

func NewGeneralCog() *GeneralCog {
	return &GeneralCog{
		InteractionCommands: subway.SetupInteractionCommandable(&subway.InteractionCommandable{}),
	}
}

type GeneralCog struct {
	InteractionCommands *subway.InteractionCommandable
}

// Assert types.

var (
	_ subway.Cog                        = (*GeneralCog)(nil)
	_ subway.CogWithInteractionCommands = (*GeneralCog)(nil)
)

func (p *GeneralCog) CogInfo() *subway.CogInfo {
	return &subway.CogInfo{
		Name:        "GeneralCog",
		Description: "General commands",
	}
}

func (p *GeneralCog) GetInteractionCommandable() *subway.InteractionCommandable {
	return p.InteractionCommands
}

func (p *GeneralCog) RegisterCog(b *subway.Subway) error {
	p.InteractionCommands.MustAddInteractionCommand(&subway.InteractionCommandable{
		Name:        "ping",
		Description: "Gets round trip API latency",
		Handler: func(ctx *subway.InteractionContext) (*discord.InteractionResponse, error) {
			now := time.Now()

			msg, err := ctx.SendFollowup(ctx.Subway.EmptySession, discord.WebhookMessageParams{
				Content: "Ping: `--- ms`",
			})
			if err != nil {
				return nil, fmt.Errorf("failed to reply: %w", err)
			}

			_, err = msg.Edit(ctx.Subway.EmptySession, discord.MessageParams{
				Content: fmt.Sprintf("Ping: `%d ms`", time.Since(now).Milliseconds()),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to edit reply: %w", err)
			}

			return &discord.InteractionResponse{
				Type: discord.InteractionCallbackTypeDeferredUpdateMessage,
				Data: nil,
			}, nil
		},
	})

	return nil
}
