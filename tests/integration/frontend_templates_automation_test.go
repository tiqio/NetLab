package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/artifact"
	"github.com/netlab/netlab/internal/app/audit"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestFrontendTemplatesImagesAutomationAndAuditUseRealServices(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:frontend-templates-automation?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	templates := storesqlite.NewTemplateRepository(database)
	digest := "sha256:" + strings.Repeat("a", 64)
	image := domain.ImageVersion{ID: domain.NewID(), RuntimeKind: domain.RuntimeQEMU, Name: "ubuntu", Version: "24.04", Digest: digest, SourceType: "local", SourceReference: "metadata-only", Format: "qcow2", SizeBytes: 1024, Availability: domain.ImageAvailable, LicenseStatus: domain.LicenseReviewed, LicenseNotes: "operator supplied", Validation: map[string]any{"format": "qcow2"}, CreatedAt: time.Now().UTC()}
	if err = templates.CreateImage(ctx, image); err != nil {
		t.Fatal(err)
	}
	template := domain.DeviceTemplate{ID: domain.NewID(), Key: "ubuntu", DisplayName: "Ubuntu", RuntimeKind: domain.RuntimeQEMU, Versions: []domain.TemplateVersion{{ID: domain.NewID(), Version: "24.04", ManifestVersion: 1, ImageVersionID: image.ID, Defaults: domain.TemplateDefaults{CPUCount: 1, MemoryMiB: 1024, Interfaces: 1, InterfaceNameFormat: "eth%d"}, Capabilities: []string{"cloud-init"}, NICDrivers: []string{"virtio-net-pci"}, ConsoleModes: []string{"telnet"}, Enabled: true}}, CreatedAt: time.Now().UTC()}
	if err = templates.UpsertTemplate(ctx, template); err != nil {
		t.Fatal(err)
	}
	catalog, err := query.NewTemplateService(templates).List(ctx)
	if err != nil || len(catalog) != 1 || catalog[0].Versions[0].ImageVersionID != image.ID {
		t.Fatalf("catalog=%+v err=%v", catalog, err)
	}

	lab, _ := command.NewLaboratoryService(topology).Create(ctx, "automation source", "", domain.RecoveryRemainStopped)
	_, _, _ = command.NewNodeService(topology).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "ubuntu", Kind: "qemu", InterfaceCount: 1, Config: map[string]any{"template_key": "ubuntu", "image_digest": digest, "password": "must-not-export"}})
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	exporter := command.NewExportService(topology, artifact.NewService(repositories, t.TempDir()))
	importer := command.NewImportService(topology, templates)
	automation := command.NewAutomationTaskService(exporter, importer, runner)
	exportTask, err := automation.Export(ctx, lab.ID, time.Hour, "frontend-export")
	if err != nil {
		t.Fatal(err)
	}
	exportTask = waitFrontendTask(t, ctx, repositories, exportTask.ID)
	if exportTask.State != domain.TaskSucceeded || exportTask.Result["artifact"] == nil {
		t.Fatalf("export task=%+v", exportTask)
	}
	bundle, err := exporter.Build(ctx, lab.ID)
	if err != nil || !bundle.Redaction.ImagesExcluded || !bundle.Redaction.CredentialsExcluded || !bundle.Redaction.BootstrapSecretsExcluded || !bundle.Redaction.CapturesExcluded {
		t.Fatalf("redacted bundle=%+v err=%v", bundle.Redaction, err)
	}
	bundle.Laboratory.Name = "automation import"
	importTask, err := automation.Import(ctx, bundle, "frontend-import")
	if err != nil {
		t.Fatal(err)
	}
	importTask = waitFrontendTask(t, ctx, repositories, importTask.ID)
	if importTask.State != domain.TaskSucceeded || importTask.Result["laboratory"] == nil {
		t.Fatalf("import task=%+v", importTask)
	}

	auditService := audit.NewService(repositories)
	if _, err = auditService.Record(ctx, "api", "laboratory.import", "laboratory", importTask.ResourceID, importTask.ID, "succeeded", "frontend-correlation", map[string]any{"token": "secret", "image_digest": digest}); err != nil {
		t.Fatal(err)
	}
	events, err := auditService.List(ctx, 10)
	if err != nil || len(events) == 0 || events[0].Details["token"] != "[REDACTED]" || events[0].Details["image_digest"] != digest {
		t.Fatalf("audit events=%+v err=%v", events, err)
	}
}

func waitFrontendTask(t *testing.T, ctx context.Context, repositories *storesqlite.Repositories, id domain.ID) domain.OperationTask {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		value, err := repositories.GetTask(ctx, id)
		if err == nil && (value.State == domain.TaskSucceeded || value.State == domain.TaskFailed || value.State == domain.TaskCancelled) {
			return value
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("task %s did not reach a terminal state", id)
	return domain.OperationTask{}
}
