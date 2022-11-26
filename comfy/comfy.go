package comfy

import (
	"github.com/WelcomerTeam/Comfy/comfy/plugins"
	sandwich "github.com/WelcomerTeam/Sandwich/sandwich"
)

func NewComfy(identifierName string, sandwichClient *sandwich.Sandwich) (bot *sandwich.Bot) {
	bot = sandwich.NewBot(sandwichClient.Logger)

	bot.MustRegisterCog(plugins.NewGeneralCog())

	return bot
}
