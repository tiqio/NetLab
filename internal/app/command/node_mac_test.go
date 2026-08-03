package command

import "testing"

func TestSequentialMACUsesStablePrefixAndSlotSuffix(t *testing.T) {
	prefix := [3]byte{0xaa, 0xbb, 0xcc}
	if got := sequentialMAC(prefix, 0); got != "02:00:aa:bb:cc:00" {
		t.Fatalf("slot 0 MAC=%q", got)
	}
	if got := sequentialMAC(prefix, 9); got != "02:00:aa:bb:cc:09" {
		t.Fatalf("slot 9 MAC=%q", got)
	}
}

func TestTemplateInterfaceNameReservesInternalControlPort(t *testing.T) {
	if name, internal := templateInterfaceName("G0/%d", 0, 1); name != "internal0" || !internal {
		t.Fatalf("slot 0 name=%q internal=%v", name, internal)
	}
	if name, internal := templateInterfaceName("G0/%d", 1, 1); name != "G0/0" || internal {
		t.Fatalf("slot 1 name=%q internal=%v", name, internal)
	}
	if name, internal := templateInterfaceName("G0/%d", 9, 1); name != "G0/8" || internal {
		t.Fatalf("slot 9 name=%q internal=%v", name, internal)
	}
}
