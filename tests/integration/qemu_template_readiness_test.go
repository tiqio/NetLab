package integration_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOperatorSuppliedQEMUTemplateMediaIsExplicit(t *testing.T) {
	media := map[string]string{"ubuntu-qemu": os.Getenv("NETLAB_UBUNTU_QCOW2"), "vyos": os.Getenv("NETLAB_VYOS_QCOW2"), "fancywan": os.Getenv("NETLAB_FANCYWAN_QCOW2")}
	validated := 0
	for family, path := range media {
		t.Run(family, func(t *testing.T) {
			if path == "" {
				t.Skipf("%s media not supplied; set the documented environment variable on the target host", family)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if !info.Mode().IsRegular() || filepath.Ext(path) != ".qcow2" {
				t.Fatalf("invalid operator media %s", path)
			}
			validated++
		})
	}
	_ = validated
}
