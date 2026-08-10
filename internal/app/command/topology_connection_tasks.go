package command

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

const (
	TopologyConnectionCreateTaskKind = "topology_connection.create"
	TopologyConnectionDeleteTaskKind = "topology_connection.delete"
)

type TopologyConnectionTaskEnvelope struct {
	Connection         domain.TopologyConnection `json:"connection"`
	Task               domain.OperationTask      `json:"task"`
	LaboratoryRevision domain.Revision           `json:"laboratory_revision"`
}

func TopologyConnectionRequestFingerprint(laboratoryID domain.ID, source, target domain.ConnectionEndpoint, config domain.TopologyConnectionConfig) string {
	body, _ := json.Marshal(map[string]any{"laboratory_id": laboratoryID, "source": source, "target": target, "config": config})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func NewTopologyConnectionOperation(kind string, resourceID domain.ID, idempotencyKey, fingerprint string, input map[string]any) domain.OperationTask {
	return domain.OperationTask{ID: domain.NewID(), Kind: kind, ResourceType: "topology_connection", ResourceID: resourceID, IdempotencyKey: idempotencyKey, RequestFingerprint: fingerprint, State: domain.TaskQueued, ProgressTotal: 2, Input: input, CreatedAt: time.Now().UTC()}
}
