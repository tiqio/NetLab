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

func TestImageCompatibilitySeparatesQEMUDeviceFamilies(t *testing.T) {
	vyos := DeviceTemplate{Key: "vyos", DisplayName: "VyOS", RuntimeKind: RuntimeQEMU}
	unbound := TemplateVersion{}
	if !ImageCompatibleWithTemplate(ImageVersion{RuntimeKind: RuntimeQEMU, Name: "VyOS", SourceReference: "vyos-rolling.qcow2"}, vyos, unbound) {
		t.Fatal("VyOS image should match the VyOS template")
	}
	if ImageCompatibleWithTemplate(ImageVersion{RuntimeKind: RuntimeQEMU, Name: "Ubuntu", SourceReference: "ubuntu-24.04.qcow2"}, vyos, unbound) {
		t.Fatal("Ubuntu image must not match the VyOS template")
	}
	if ImageCompatibleWithTemplate(ImageVersion{RuntimeKind: RuntimeQEMU, Name: "FortiGate", SourceReference: "fortinet-FGT.qcow2"}, vyos, unbound) {
		t.Fatal("FortiGate image must not match the VyOS template")
	}
}

func TestBoundTemplateVersionOnlyAcceptsAssignedImage(t *testing.T) {
	template := DeviceTemplate{Key: "ubuntu-qemu", DisplayName: "Ubuntu", RuntimeKind: RuntimeQEMU}
	version := TemplateVersion{ImageVersionID: "ubuntu-24"}
	if !ImageCompatibleWithTemplate(ImageVersion{ID: "ubuntu-24", RuntimeKind: RuntimeQEMU, Name: "Ubuntu"}, template, version) {
		t.Fatal("assigned image should be compatible")
	}
	if ImageCompatibleWithTemplate(ImageVersion{ID: "ubuntu-22", RuntimeKind: RuntimeQEMU, Name: "Ubuntu"}, template, version) {
		t.Fatal("a bound template version must reject other images from the same family")
	}
}

func TestImageCompatibilitySeparatesDockerDeviceFamilies(t *testing.T) {
	nginx := DeviceTemplate{Key: "nginx-container", DisplayName: "Nginx", RuntimeKind: RuntimeDocker}
	version := TemplateVersion{RuntimeOptions: map[string]any{"recommended_image_name": "nginx"}}
	if !ImageCompatibleWithTemplate(ImageVersion{RuntimeKind: RuntimeDocker, Name: "nginx", SourceReference: "nginx:1.30-alpine"}, nginx, version) {
		t.Fatal("Nginx image should match the Nginx template")
	}
	if ImageCompatibleWithTemplate(ImageVersion{RuntimeKind: RuntimeDocker, Name: "busybox", SourceReference: "busybox:1.38.0"}, nginx, version) {
		t.Fatal("BusyBox image must not match the Nginx template")
	}
}
