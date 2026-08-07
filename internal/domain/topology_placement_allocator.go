package domain

import "fmt"

type TopologyPlacementAllocator struct {
	Footprints map[PlacementFootprintClass]PlacementFootprint
	MaxRings   int
}

func NewTopologyPlacementAllocator() TopologyPlacementAllocator {
	return TopologyPlacementAllocator{Footprints: map[PlacementFootprintClass]PlacementFootprint{
		FootprintNodeStandard:          {Class: FootprintNodeStandard, Width: 128, Height: 92, ClearanceX: 22, ClearanceY: 18},
		FootprintNodeWide:              {Class: FootprintNodeWide, Width: 168, Height: 104, ClearanceX: 24, ClearanceY: 20},
		FootprintNetworkObjectStandard: {Class: FootprintNetworkObjectStandard, Width: 132, Height: 96, ClearanceX: 22, ClearanceY: 18},
		FootprintNetworkObjectWide:     {Class: FootprintNetworkObjectWide, Width: 176, Height: 108, ClearanceX: 24, ClearanceY: 20},
	}, MaxRings: 128}
}

func (a TopologyPlacementAllocator) Allocate(resourceType PlacementResourceType, intent *PlacementIntent, occupied []PlacementOccupancy) (PlacementAssignment, error) {
	if err := ValidatePlacementIntent(resourceType, intent); err != nil {
		return PlacementAssignment{}, err
	}
	class := DefaultPlacementFootprintClass(resourceType)
	center := PlacementPoint{}
	reason := PlacementReasonDefaultAnchor
	var requested *PlacementPoint
	if intent != nil {
		if intent.FootprintClass != "" {
			class = intent.FootprintClass
		}
		if intent.PreferredX != nil {
			center = PlacementPoint{X: *intent.PreferredX, Y: *intent.PreferredY}
			copy := center
			requested = &copy
			reason = PlacementReasonPreferredAvailable
		}
	}
	footprint, ok := a.Footprints[class]
	if !ok {
		return PlacementAssignment{}, fmt.Errorf("unsupported footprint class %q", class)
	}
	stepX := footprint.Width + footprint.ClearanceX*2
	stepY := footprint.Height + footprint.ClearanceY*2
	for ring := 0; ring <= a.MaxRings; ring++ {
		for _, offset := range squareRing(ring) {
			candidate := PlacementPoint{X: center.X + float64(offset[0])*stepX, Y: center.Y + float64(offset[1])*stepY}
			if candidate.X < -MaxPlacementCoordinate || candidate.X > MaxPlacementCoordinate || candidate.Y < -MaxPlacementCoordinate || candidate.Y > MaxPlacementCoordinate {
				continue
			}
			if !a.collides(candidate, footprint, occupied) {
				adjusted := candidate != center
				if adjusted {
					reason = PlacementReasonCollisionAvoided
				}
				return PlacementAssignment{RequestedCenter: requested, AssignedCenter: candidate, Adjusted: adjusted, Reason: reason, FootprintClass: class, AlgorithmVersion: PlacementAlgorithmVersion}, nil
			}
		}
	}
	return PlacementAssignment{}, fmt.Errorf("no placement candidate available")
}

func squareRing(ring int) [][2]int {
	if ring == 0 {
		return [][2]int{{0, 0}}
	}
	result := make([][2]int, 0, ring*8)
	for x := -ring; x <= ring; x++ {
		result = append(result, [2]int{x, -ring})
	}
	for y := -ring + 1; y <= ring; y++ {
		result = append(result, [2]int{ring, y})
	}
	for x := ring - 1; x >= -ring; x-- {
		result = append(result, [2]int{x, ring})
	}
	for y := ring - 1; y > -ring; y-- {
		result = append(result, [2]int{-ring, y})
	}
	return result
}

func (a TopologyPlacementAllocator) collides(candidate PlacementPoint, footprint PlacementFootprint, occupied []PlacementOccupancy) bool {
	for _, value := range occupied {
		existing, ok := a.Footprints[value.FootprintClass]
		if !ok {
			existing = a.Footprints[DefaultPlacementFootprintClass(PlacementNode)]
		}
		dx := abs(candidate.X - value.X)
		dy := abs(candidate.Y - value.Y)
		if dx < footprint.Width/2+existing.Width/2+max(footprint.ClearanceX, existing.ClearanceX) && dy < footprint.Height/2+existing.Height/2+max(footprint.ClearanceY, existing.ClearanceY) {
			return true
		}
	}
	return false
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
func max(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
