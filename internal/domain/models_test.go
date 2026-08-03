package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInterfaceOperationalStateUsesContractProperty(t *testing.T) {
	body, err := json.Marshal(Interface{OperationalState: "up"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"operational_state":"up"`) {
		t.Fatalf("unexpected interface payload: %s", body)
	}
	if strings.Contains(string(body), `"oper_state"`) {
		t.Fatalf("database column name leaked into API payload: %s", body)
	}
}
