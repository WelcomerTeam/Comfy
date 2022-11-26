package comfy

import (
	"context"
	"fmt"

	"github.com/WelcomerTeam/Comfy/comfy-interactions/plugins"
	subway "github.com/WelcomerTeam/Subway/subway"
)

func NewComfy(ctx context.Context, identifierName string, options subway.SubwayOptions) *subway.Subway {
	subway, err := subway.NewSubway(ctx, options)
	if err != nil {
		panic(fmt.Errorf("failed to create subway client. subway.NewClient(%v): %w", options, err))
	}

	subway.MustRegisterCog(plugins.NewGeneralCog())
	subway.MustRegisterCog(plugins.NewPogCog())

	return subway
}
