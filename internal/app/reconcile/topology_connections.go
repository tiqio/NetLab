package reconcile

import (
	"github.com/netlab/netlab/internal/app/command"
)

func NewUnifiedTopologyConnectionService(repository command.TopologyConnectionRepository, links *command.TopologyTaskService, networks *NetworkObjectTaskService) *command.TopologyConnectionService {
	return command.NewTopologyConnectionService(repository, links, networks)
}
