package rancher

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Component discovers the container images for one part of a Rancher
// release (e.g. rancher-manager, fleet, turtles).
type Component interface {
	Name() string
	Images(ctx context.Context, rc *Context) ([]string, error)
}

// logFields tags a log entry with the source file and field a Component is
// currently reading, alongside the component name already added by
// Context.Log().
func logFields(file, field string) logrus.Fields {
	return logrus.Fields{"file": file, "field": field}
}

// components lists every component of the rancher subsystem, in the order
// they run. Adding a new component means adding it here — there is no
// implicit init()-based self-registration.
var components = []Component{
	managerComponent{},
	webhookComponent{},
	fleetComponent{},
	shellComponent{},
	turtlesComponent{},
	sucComponent{},
	systemAgentComponent{},
}
