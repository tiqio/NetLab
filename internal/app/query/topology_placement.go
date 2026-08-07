package query

import (
	"context"
	"math"
	"sort"

	"github.com/netlab/netlab/internal/domain"
)

type TopologyPlacementReader interface {
	ListPlacements(context.Context, domain.ID) ([]domain.TopologyPlacement, error)
}

type TopologyPlacementService struct{ reader TopologyPlacementReader }

func NewTopologyPlacementService(reader TopologyPlacementReader) *TopologyPlacementService {
	return &TopologyPlacementService{reader: reader}
}

func (s *TopologyPlacementService) List(ctx context.Context, laboratoryID domain.ID) ([]domain.TopologyPlacement, error) {
	placements, err := s.reader.ListPlacements(ctx, laboratoryID)
	if err != nil {
		return nil, err
	}
	sort.Slice(placements, func(left, right int) bool { return placements[left].ResourceID < placements[right].ResourceID })
	return placements, nil
}

func ResolveTopologyPlacements(snapshot domain.TopologySnapshot) []domain.TopologyPlacement {
	resolved := append([]domain.TopologyPlacement(nil), snapshot.Placements...)
	present := make(map[domain.ID]bool, len(resolved))
	occupied := make([]placementPoint, 0, len(snapshot.Nodes)+len(snapshot.NetworkObjects))
	for _, placement := range resolved {
		present[placement.ResourceID] = true
		occupied = append(occupied, placementPoint{x: placement.X, y: placement.Y})
	}
	type resource struct {
		id   domain.ID
		kind domain.PlacementResourceType
	}
	resources := make([]resource, 0, len(snapshot.Nodes)+len(snapshot.NetworkObjects))
	for _, node := range snapshot.Nodes {
		resources = append(resources, resource{id: node.ID, kind: domain.PlacementNode})
	}
	for _, object := range snapshot.NetworkObjects {
		resources = append(resources, resource{id: object.ID, kind: domain.PlacementNetworkObject})
	}
	sort.Slice(resources, func(left, right int) bool { return resources[left].id < resources[right].id })
	for _, value := range resources {
		if present[value.id] {
			continue
		}
		point := deterministicPlacement(value.id, occupied)
		occupied = append(occupied, point)
		resolved = append(resolved, domain.TopologyPlacement{LaboratoryID: snapshot.Laboratory.ID, ResourceID: value.id, ResourceType: value.kind, X: point.x, Y: point.y})
	}
	sort.Slice(resolved, func(left, right int) bool { return resolved[left].ResourceID < resolved[right].ResourceID })
	return resolved
}

type placementPoint struct{ x, y float64 }

func deterministicPlacement(id domain.ID, occupied []placementPoint) placementPoint {
	var hash int32
	for _, character := range string(id) {
		hash = hash*31 + int32(character)
	}
	seed := int64(hash)
	if seed < 0 {
		seed = -seed
	}
	angle := float64(seed%360) * math.Pi / 180
	radius := float64(100 + (seed%5)*55)
	for attempt := 0; attempt < 50; attempt++ {
		point := placementPoint{x: math.Round(math.Cos(angle) * radius), y: math.Round(math.Sin(angle) * radius)}
		available := true
		for _, current := range occupied {
			if math.Hypot(current.x-point.x, current.y-point.y) <= 80 {
				available = false
				break
			}
		}
		if available {
			return point
		}
		angle += .75
		radius += 10
	}
	return placementPoint{x: radius, y: radius}
}
