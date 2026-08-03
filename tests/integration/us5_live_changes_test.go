package integration

import (
	"os"
	"testing"
)

func TestPrivilegedRunningNodeMutations(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED") != "1" {
		t.Skip("set NETLAB_PRIVILEGED=1 on acceptance host")
	}
	t.Skip("requires a running operator-supplied QEMU image")
}
