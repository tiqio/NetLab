package integration

import (
	"os"
	"testing"
)

func TestPrivilegedThreeHopCaptureAndPathObservation(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED") != "1" {
		t.Skip("set NETLAB_PRIVILEGED=1 on the acceptance host")
	}
	t.Skip("requires acceptance topology images and is executed by the quickstart validation script")
}
