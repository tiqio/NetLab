package compliance

import (
	"strings"
	"testing"
)

func TestValidateRuntimeAuthorityRejectsSecondExternalNetLabListener(t *testing.T) {
	listeners := ParseSocketListeners(strings.NewReader(`
State Recv-Q Send-Q Local Address:Port Peer Address:Port Process
LISTEN 0 4096 *:18082 *:* users:(("netlabd",pid=200,fd=10))
LISTEN 0 4096 *:18080 *:* users:(("netlabd",pid=100,fd=11))
`))
	if err := ValidateRuntimeAuthority(listeners, "18082"); err == nil {
		t.Fatal("expected a second externally reachable NetLab listener to be rejected")
	}
}

func TestValidateRuntimeAuthorityAllowsLoopbackValidationListener(t *testing.T) {
	listeners := ParseSocketListeners(strings.NewReader(`
State Recv-Q Send-Q Local Address:Port Peer Address:Port Process
LISTEN 0 4096 *:18082 *:* users:(("netlabd",pid=200,fd=10))
LISTEN 0 4096 127.0.0.1:18080 0.0.0.0:* users:(("netlabd",pid=100,fd=11))
`))
	if err := ValidateRuntimeAuthority(listeners, "18082"); err != nil {
		t.Fatal(err)
	}
}
