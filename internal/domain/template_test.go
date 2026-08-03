package domain

import "testing"

func TestTemplateValidationAndCapabilities(t *testing.T) {
	template := DeviceTemplate{Key: "ubuntu-qemu", DisplayName: "Ubuntu", RuntimeKind: RuntimeQEMU, Versions: []TemplateVersion{{Version: "24.04", ManifestVersion: 1, Defaults: TemplateDefaults{CPUCount: 2, MemoryMiB: 2048, Interfaces: 1}, NICDrivers: []string{"virtio-net-pci"}, Capabilities: []string{"cloud_init"}}}}
	if err := template.Validate(); err != nil {
		t.Fatal(err)
	}
	if !template.Versions[0].HasCapability("cloud_init") {
		t.Fatal("capability missing")
	}
}
func TestImageStartGate(t *testing.T) {
	image := ImageVersion{Digest: "sha256:abc", Availability: ImageAvailable, LicenseStatus: LicenseUnreviewed}
	if image.CanStart() == nil {
		t.Fatal("expected license gate")
	}
	image.LicenseStatus = LicenseReviewed
	if err := image.CanStart(); err != nil {
		t.Fatal(err)
	}
}
