package plugins

import (
	"fmt"
	"github.com/WelcomerTeam/Discord/discord"
	sandwich "github.com/WelcomerTeam/Sandwich/sandwich"
	"golang.org/x/xerrors"
	"time"
)

func NewGeneralCog() *GeneralCog {
	return &GeneralCog{
		Commands:            sandwich.SetupCommandable(&sandwich.Commandable{}),
		InteractionCommands: sandwich.SetupInteractionCommandable(&sandwich.InteractionCommandable{}),
	}
}

type GeneralCog struct {
	Commands            *sandwich.Commandable
	InteractionCommands *sandwich.InteractionCommandable
}

// Assert types.

var (
	_ sandwich.Cog             = (*GeneralCog)(nil)
	_ sandwich.CogWithCommands = (*GeneralCog)(nil)
)

func (p *GeneralCog) CogInfo() *sandwich.CogInfo {
	return &sandwich.CogInfo{
		Name:        "GeneralCog",
		Description: "General commands",
	}
}

func (p *GeneralCog) GetCommandable() *sandwich.Commandable {
	return p.Commands
}

func (p *GeneralCog) RegisterCog(b *sandwich.Bot) error {
	p.Commands.MustAddCommand(&sandwich.Commandable{
		Name:        "ping",
		Description: "Gets round trip API latency",
		Handler: func(ctx *sandwich.CommandContext) (err error) {
			now := time.Now()

			msg, err := ctx.Reply(ctx.EventContext.Session, discord.MessageParams{
				Content: "Ping: `--- ms`",
			})
			if err != nil {
				return xerrors.Errorf("Failed to reply: %v", err.Error())
			}

			_, err = msg.Edit(ctx.EventContext.Session, discord.MessageParams{
				Content: fmt.Sprintf("Ping: `%d ms`", time.Since(now).Milliseconds()),
			})
			if err != nil {
				return xerrors.Errorf("Failed to reply: %v", err.Error())
			}

			return
		},
	})

	return nil
}
