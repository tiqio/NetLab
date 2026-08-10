package reconcile

import (
	"github.com/netlab/netlab/internal/app/command"
)

func NewUnifiedTopologyConnectionService(repository command.TopologyConnectionRepository, links command.TopologyConnectionLinkOperations, networks command.TopologyConnectionNetworkOperations) *command.TopologyConnectionService {
	return command.NewTopologyConnectionService(repository, links, networks)
}
