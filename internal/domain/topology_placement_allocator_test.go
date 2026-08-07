package domain

import "testing"

func TestAllocateTopologyPlacementUsesPreferredAndAvoidsCollisions(t *testing.T) {
	allocator := NewTopologyPlacementAllocator()
	x, y := 0.0, 0.0
	first, err := allocator.Allocate(PlacementNode, &PlacementIntent{PreferredX: &x, PreferredY: &y}, nil)
	if err != nil || first.AssignedCenter.X != 0 || first.AssignedCenter.Y != 0 || first.Adjusted {
		t.Fatalf("unexpected first assignment: %#v %v", first, err)
	}
	existing := []PlacementOccupancy{{ResourceID: "existing", X: first.AssignedCenter.X, Y: first.AssignedCenter.Y, FootprintClass: first.FootprintClass}}
	second, err := allocator.Allocate(PlacementNode, &PlacementIntent{PreferredX: &x, PreferredY: &y}, existing)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Adjusted || second.Reason != PlacementReasonCollisionAvoided || second.AssignedCenter == first.AssignedCenter {
		t.Fatalf("expected collision avoidance: %#v", second)
	}
}

func TestAllocateTopologyPlacementIsStableForTwentyResources(t *testing.T) {
	allocator := NewTopologyPlacementAllocator()
	var occupied []PlacementOccupancy
	seen := map[PlacementPoint]struct{}{}
	for index := 0; index < 20; index++ {
		assignment, err := allocator.Allocate(PlacementNetworkObject, nil, occupied)
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := seen[assignment.AssignedCenter]; exists {
			t.Fatalf("duplicate assignment %#v", assignment.AssignedCenter)
		}
		seen[assignment.AssignedCenter] = struct{}{}
		occupied = append(occupied, PlacementOccupancy{ResourceID: ID(string(rune('a' + index))), X: assignment.AssignedCenter.X, Y: assignment.AssignedCenter.Y, FootprintClass: assignment.FootprintClass})
	}
}

func TestAllocateTopologyPlacementHonorsClearanceAndExhaustion(t *testing.T) {
	allocator := NewTopologyPlacementAllocator()
	x, y := 10.0, 20.0
	first, err := allocator.Allocate(PlacementNode, &PlacementIntent{PreferredX: &x, PreferredY: &y, FootprintClass: FootprintNodeWide}, nil)
	if err != nil {
		t.Fatal(err)
	}
	occupied := []PlacementOccupancy{{ResourceID: "existing", X: first.AssignedCenter.X, Y: first.AssignedCenter.Y, FootprintClass: FootprintNodeWide}}
	second, err := allocator.Allocate(PlacementNode, &PlacementIntent{PreferredX: &x, PreferredY: &y, FootprintClass: FootprintNodeWide}, occupied)
	if err != nil {
		t.Fatal(err)
	}
	if second.AssignedCenter.X == first.AssignedCenter.X && second.AssignedCenter.Y == first.AssignedCenter.Y {
		t.Fatal("clearance did not move the colliding placement")
	}
	allocator.MaxRings = 0
	if _, err = allocator.Allocate(PlacementNode, &PlacementIntent{PreferredX: &x, PreferredY: &y, FootprintClass: FootprintNodeWide}, occupied); err == nil {
		t.Fatal("expected candidate exhaustion")
	}
}
