package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	httpapi "github.com/netlab/netlab/internal/api/http"
	"github.com/netlab/netlab/internal/api/mcp"
	"github.com/netlab/netlab/internal/api/stream"
	"github.com/netlab/netlab/internal/app/artifact"
	"github.com/netlab/netlab/internal/app/audit"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/events"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	cgroupRuntime "github.com/netlab/netlab/internal/runtime/cgroup"
	consoleRuntime "github.com/netlab/netlab/internal/runtime/console"
	dockerruntime "github.com/netlab/netlab/internal/runtime/docker"
	fortigateRuntime "github.com/netlab/netlab/internal/runtime/fortigate"
	imageRuntime "github.com/netlab/netlab/internal/runtime/image"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
	qemuruntime "github.com/netlab/netlab/internal/runtime/qemu"
	credentialstore "github.com/netlab/netlab/internal/store/credential"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
	"github.com/netlab/netlab/internal/support/config"
	"github.com/netlab/netlab/internal/support/observability"
	"github.com/netlab/netlab/internal/support/readiness"
)

var (
	version        = "dev"
	candidateID    = "dev"
	binaryDigest   string
	contractDigest string
	builtAt        string
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "release" {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"version": version, "candidate_id": candidateID, "binary_digest": binaryDigest, "contract_digest": contractDigest, "built_at": builtAt})
		return
	}
	validateOnly := len(os.Args) > 1 && os.Args[1] == "validate-config"
	if validateOnly {
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}
	configPath := flag.String("config", "", "configuration file")
	flag.Parse()
	cfg, err := config.LoadRaw(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load config:", err)
		os.Exit(1)
	}
	applyBuildIdentity(&cfg)
	if err = cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "validate config:", err)
		os.Exit(1)
	}
	if validateOnly {
		return
	}
	releaseIdentity := configuredReleaseIdentity(cfg)
	logger := observability.NewLogger(slog.LevelInfo)
	if warning := cfg.SecurityWarning(); warning != "" {
		logger.Warn(warning)
	}
	if err = os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o700); err != nil {
		logger.Error("create state directory", "error", err)
		os.Exit(1)
	}
	database, err := storesqlite.Open(context.Background(), cfg.DatabasePath)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topologyRepository := storesqlite.NewTopologyRepository(database)
	templateRepository := storesqlite.NewTemplateRepository(database)
	taskRunner := task.NewRunner(repositories, 4, 256)
	defer taskRunner.Close()
	publisher := events.NewPublisher(repositories)
	lock, err := reconcile.AcquireInstanceLock(filepath.Join(cfg.RuntimeDir, "netlab.lock"))
	if err != nil {
		_ = os.MkdirAll(cfg.RuntimeDir, 0o700)
		lock, err = reconcile.AcquireInstanceLock(filepath.Join(cfg.RuntimeDir, "netlab.lock"))
	}
	if err != nil {
		logger.Error("acquire instance lock", "error", err)
		os.Exit(1)
	}
	defer lock.Close()
	metrics := &observability.Metrics{}
	server := httpapi.NewServer(cfg.Listen, logger, metrics)
	auditService := audit.NewService(repositories)
	ownershipQueries := query.NewRuntimeOwnershipService(repositories)
	capabilityQueries := query.NewRuntimeCapabilityService(repositories)
	idempotency := command.NewIdempotencyService(repositories, 24*time.Hour)
	labCommands := command.NewLaboratoryService(topologyRepository)
	labQueries := query.NewLaboratoryService(topologyRepository)
	nodeCommands := command.NewNodeService(topologyRepository, templateRepository)
	var seedManager *qemuruntime.SeedManager
	if seedManager, err = qemuruntime.NewSeedManager(cfg.StateDir); err == nil {
		nodeCommands.SetSeedBuilder(seedManager)
	} else {
		logger.Warn("cloud-init seed builder unavailable", "error", err)
	}
	consoleCredentials := consoleRuntime.NewCredentialStore(cfg.StateDir, seedManager)
	var nodeCredentialStore *credentialstore.Store
	var nodeCredentialStoreErr error
	if nodeCredentialStore, nodeCredentialStoreErr = credentialstore.Open(cfg.Credentials.DatabasePath, cfg.Credentials.MasterKeyPath); nodeCredentialStoreErr != nil {
		logger.Warn("FortiGate credential store unavailable", "error", nodeCredentialStoreErr)
	} else {
		defer nodeCredentialStore.Close()
	}
	fortiGateCredentials := command.NewFortiGateCredentialService(topologyRepository, nodeCredentialStore, nodeCredentialStoreErr, fortigateRuntime.NewConsole(cfg.RuntimeDir), taskRunner)
	linkCommands := command.NewLinkService(topologyRepository)
	placementCommands := command.NewTopologyPlacementService(topologyRepository)
	topologyTasks := command.NewTopologyTaskService(topologyRepository, taskRunner)
	linkReconnectTasks := command.NewLinkReconnectService(topologyRepository, taskRunner)
	laboratoryTasks := command.NewLaboratoryTaskService(topologyRepository, taskRunner)
	server.Engine().Use(httpapi.MutationAutomation(idempotency, repositories, auditService))
	httpapi.NewTopologyHandlers(labCommands, labQueries, nodeCommands, linkCommands, repositories, idempotency, topologyTasks, laboratoryTasks).Register(server.Engine())
	httpapi.NewTopologyPlacementHandlers(placementCommands).Register(server.Engine())
	httpapi.NewLinkReconnectHandlers(linkReconnectTasks).Register(server.Engine())
	httpapi.NewFortiGateCredentialHandlers(fortiGateCredentials, cfg.Deployment.ManagementScopes).Register(server.Engine())
	templateQueries := query.NewTemplateService(templateRepository)
	templateDirectory := cfg.TemplateDir
	if _, statErr := os.Stat(templateDirectory); statErr != nil {
		if _, localErr := os.Stat("templates"); localErr == nil {
			templateDirectory = "templates"
		}
	}
	if err = templateQueries.LoadBuiltins(context.Background(), templateDirectory); err != nil {
		logger.Error("load built-in templates", "error", err)
		os.Exit(1)
	}
	readinessPath := cfg.TemplateReadinessPath
	if _, statErr := os.Stat(readinessPath); statErr != nil {
		if _, localErr := os.Stat("compliance/template-readiness.json"); localErr == nil {
			readinessPath = "compliance/template-readiness.json"
		}
	}
	if err = templateQueries.LoadReadinessForCandidate(readinessPath, releaseIdentity.CandidateID); err != nil {
		if cfg.Deployment.Role == "authoritative" {
			logger.Error("template readiness validation failed", "error", err)
			os.Exit(1)
		}
		logger.Warn("template readiness unavailable", "error", err)
	}
	nodeCommands.SetTemplateReadinessResolver(templateQueries)
	httpapi.NewTemplateHandlers(templateQueries, templateRepository, imageRuntime.NewImporter(cfg.StateDir), consoleCredentials).Register(server.Engine())
	artifactService := artifact.NewService(repositories, cfg.StateDir)
	taskQueries := query.NewTaskService(repositories, taskRunner)
	exportService := command.NewExportService(topologyRepository, artifactService)
	importService := command.NewImportService(topologyRepository, templateRepository)
	automationTasks := command.NewAutomationTaskService(exportService, importService, taskRunner)
	trafficWorkloads := command.NewTrafficWorkloadService(repositories, taskRunner)
	httpapi.NewArtifactHandlers(artifactService).Register(server.Engine())
	httpapi.NewClientToolsHandlers(cfg.StateDir).Register(server.Engine())
	automationHandlers := httpapi.NewAutomationHandlers(taskQueries, exportService, importService, automationTasks, auditService)
	automationHandlers.SetReleaseService(query.NewReleaseService(releaseIdentity))
	automationHandlers.Register(server.Engine())
	httpapi.NewRuntimeOwnershipHandlers(ownershipQueries).Register(server.Engine())
	httpapi.NewNodeCapabilityHandlers(capabilityQueries).Register(server.Engine())
	consoleLimits := consoleRuntime.Limits{IdleTimeout: 30 * time.Minute, BytesPerSecond: 8 << 20, MaximumSession: 8 * time.Hour}
	consoleHandlers := stream.NewConsoleHandlers(filepath.Join(cfg.StateDir, "runtime", "qemu"), consoleLimits, topologyRepository)
	var nodeAddressResolver *consoleRuntime.SSHBackend
	if seedManager != nil {
		nodeAddressResolver = consoleRuntime.NewSSHBackend(topologyRepository, consoleCredentials)
		consoleHandlers.SetSSHConsole(nodeAddressResolver)
	}
	consoleHandlers.SetCredentialSource(consoleCredentials)
	consoleHandlers.Register(server.Engine())
	captureManager := reconcile.NewCaptureManager(cfg.StateDir, cfg.Captures.Concurrent, cfg.Captures.GlobalMaxBytes, cfg.Captures.Retention, artifactService)
	captureManager.SetNetworkObjectRepository(repositories)
	trafficFilterManager := reconcile.NewTrafficFilterManager(captureManager)
	captureTasks := reconcile.NewCaptureTaskService(captureManager, trafficFilterManager, taskRunner)
	captureManager.SetObserver(trafficFilterManager.ObserveManagedCapture)
	httpapi.NewCaptureHandlers(captureManager, trafficFilterManager, captureTasks).Register(server.Engine())
	httpapi.NewTrafficWorkloadHandlers(trafficWorkloads).Register(server.Engine())
	pcRuntime, pcRuntimeErr := linuxnet.NewPCRuntime(nil)
	if pcRuntimeErr != nil {
		logger.Warn("PC runtime unavailable", "error", pcRuntimeErr)
	}
	var networkObjectConsoleHandlers *stream.NetworkObjectConsoleHandlers
	if pcRuntime != nil {
		networkObjectConsoleHandlers = stream.NewNetworkObjectConsoleHandlers(repositories, pcRuntime, consoleLimits)
		networkObjectConsoleHandlers.Register(server.Engine())
	}
	bridgeRuntime, bridgeRuntimeErr := linuxnet.NewBridgeRuntime(nil)
	if bridgeRuntimeErr != nil {
		logger.Warn("bridge runtime unavailable", "error", bridgeRuntimeErr)
	}
	natRuntime, natRuntimeErr := linuxnet.NewNATRuntime(nil)
	if natRuntimeErr != nil {
		logger.Warn("NAT runtime unavailable", "error", natRuntimeErr)
	} else if manager, managerErr := linuxnet.NewDNSMasqManager(cfg.StateDir); managerErr == nil {
		natRuntime.SetDHCPManager(manager)
	} else {
		logger.Warn("NAT DHCP/RA helper unavailable", "error", managerErr)
	}
	switchL2Runtime, switchL2RuntimeErr := linuxnet.NewSwitchL2Runtime(nil)
	if switchL2RuntimeErr != nil {
		logger.Warn("L2 switch runtime unavailable", "error", switchL2RuntimeErr)
	}
	switchL3Runtime, switchL3RuntimeErr := linuxnet.NewSwitchL3Runtime(nil)
	if switchL3RuntimeErr != nil {
		logger.Warn("L3 switch runtime unavailable", "error", switchL3RuntimeErr)
	}
	networkRuntimes := reconcile.NetworkRuntimeDispatch{Bridge: bridgeRuntime, NAT: natRuntime, PC: pcRuntime, SwitchL2: switchL2Runtime, SwitchL3: switchL3Runtime}
	networkService := reconcile.NewNetworkObjectService(repositories, networkRuntimes)
	networkService.AddObjectLinkObserverCleanup(captureManager)
	networkService.AddObjectLinkObserverCleanup(trafficFilterManager)
	networkTasks := reconcile.NewNetworkObjectTaskService(networkService, taskRunner)
	httpapi.NewNetworkHandlers(networkService, networkTasks, pcRuntime, bridgeRuntime, natRuntime, switchL2Runtime, switchL3Runtime).Register(server.Engine())
	topologyConnections := reconcile.NewUnifiedTopologyConnectionService(repositories, topologyTasks, networkTasks)
	httpapi.NewTopologyConnectionHandlers(topologyConnections, repositories).Register(server.Engine())
	server.Engine().GET("/api/v1/events", gin.WrapH(stream.NewEventHandler(publisher)))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	captureManager.StartCleanup(ctx, time.Minute)
	var qemuAdapter *qemuruntime.Adapter
	if qemuAdapter, err = qemuruntime.NewAdapter(cfg.StateDir); err != nil {
		logger.Warn("qemu runtime unavailable", "error", err)
	}
	if qemuAdapter != nil {
		topologyTasks.SetNodeDeletionRuntime("qemu", qemuAdapter)
	}
	linkRuntime, linkRuntimeErr := linuxnet.NewLinkRuntime(nil)
	if linkRuntimeErr != nil {
		logger.Warn("live link runtime unavailable", "error", linkRuntimeErr)
	}
	portMapper, portMapperErr := linuxnet.NewPortMapper(nil)
	if portMapperErr != nil {
		logger.Warn("port mapping runtime unavailable", "error", portMapperErr)
	}
	var interfaceCommands *command.InterfaceService
	var guestCommands *command.GuestCommandService
	var portMappingCommands *command.PortMappingService
	if qemuAdapter != nil && linkRuntime != nil {
		interfaceCommands = command.NewInterfaceService(topologyRepository, taskRunner, qemuAdapter, linkRuntime, []string{"virtio-net-pci", "e1000", "e1000e", "vmxnet3"})
		guestCommands = command.NewGuestCommandService(topologyRepository, taskRunner, qemuAdapter, auditService)
		guestCommands.SetCapabilityRepository(repositories)
	}
	if portMapper != nil {
		portMappingCommands = command.NewPortMappingService(repositories, taskRunner, portMapper)
		if nodeAddressResolver != nil {
			portMappingCommands.SetAutoResolver(topologyRepository, nodeAddressResolver)
		}
	}
	deviceReadiness := query.NewDeviceReadinessService(topologyRepository, topologyRepository, repositories, repositories)
	mcpTools := mcp.Tools(mcp.Services{Labs: labCommands, LabQueries: labQueries, Templates: templateQueries, Nodes: nodeCommands, NodeSettings: topologyRepository, Links: linkCommands, TopologyOps: topologyTasks, LabOps: laboratoryTasks, Interfaces: interfaceCommands, Guest: guestCommands, Mappings: portMappingCommands, Tasks: taskQueries, Exporter: exportService, Importer: importService, Automation: automationTasks, Captures: captureManager, Filters: trafficFilterManager, CaptureOps: captureTasks, Capabilities: capabilityQueries, DeviceReadiness: deviceReadiness, ConsoleIdle: consoleLimits.IdleTimeout})
	mcpTools = append(mcpTools, mcp.TrafficWorkloadTools(trafficWorkloads)...)
	mcpTools = append(mcpTools, mcp.NetworkTools(networkService, networkTasks)...)
	mcpTools = append(mcpTools, mcp.TopologyConnectionTools(topologyConnections, repositories)...)
	mcpTools = append(mcpTools, mcp.TopologyPlacementTools(placementCommands)...)
	mcpTools = append(mcpTools, mcp.LinkReconnectTools(linkReconnectTasks)...)
	mcp.NewServer(mcpTools, idempotency, auditService).Register(server.Engine())
	cgroupManager := cgroupRuntime.NewManager("")
	if prepareErr := cgroupManager.Prepare(ctx); prepareErr != nil {
		logger.Warn("cgroup resource enforcement unavailable until retry", "error", prepareErr)
	}
	resourceManager := reconcile.NewResourceManager(qemuAdapter, cgroupManager, cfg.StateDir)
	resourceManager.SetMaxRunningQEMU(cfg.Resources.MaxRunningQEMU)
	if linkRuntime != nil {
		topologyTasks.SetNodeDeletionCleanup(resourceManager, linkRuntime)
	} else {
		topologyTasks.SetNodeDeletionCleanup(resourceManager, nil)
	}
	nodeOperationHandlers := httpapi.NewNodeOperationsHandlers(interfaceCommands, guestCommands, portMappingCommands, topologyRepository, resourceManager, seedManager)
	nodeOperationHandlers.SetNodeCredentialReader(consoleCredentials)
	nodeOperationHandlers.SetDeviceReadiness(deviceReadiness)
	nodeOperationHandlers.Register(server.Engine())
	if endpointRuntime, runtimeErr := linuxnet.NewEndpointRuntime(); runtimeErr == nil {
		var dockerAdapter *dockerruntime.Adapter
		if dockerAdapter, runtimeErr = dockerruntime.NewAdapter(); runtimeErr != nil {
			logger.Warn("docker runtime unavailable", "error", runtimeErr)
		}
		if dockerAdapter != nil {
			topologyTasks.SetNodeDeletionRuntime("docker", dockerAdapter)
			consoleHandlers.SetDockerConsole(dockerAdapter)
			nodeOperationHandlers.SetNetworkDiagnostics(dockerAdapter)
		}
		workloadRuntime := linuxnet.NewTrafficWorkloadRuntime(nil, nil, func(execCtx context.Context, node domain.Node, argv []string, timeout time.Duration, outputLimit int) (linuxnet.TrafficWorkloadGuestResult, error) {
			if qemuAdapter == nil {
				return linuxnet.TrafficWorkloadGuestResult{}, domain.Problem{Code: domain.ProblemCodeCapabilityUnavailable, Message: "QEMU guest execution is unavailable", Retryable: true, ResourceType: "node", ResourceID: node.ID}
			}
			result, execErr := qemuAdapter.GuestExec(execCtx, node, qemuruntime.GuestExecRequest{Argv: argv, Timeout: timeout, OutputLimit: outputLimit})
			return linuxnet.TrafficWorkloadGuestResult{ExitCode: result.ExitCode, Stdout: result.Stdout, Stderr: result.Stderr, Truncated: result.Truncated}, execErr
		})
		workloadResolver := linuxnet.NewTrafficWorkloadTargetResolver(topologyRepository, repositories)
		workloadReconciler := reconcile.NewTrafficWorkloadReconciler(repositories, workloadResolver, workloadRuntime, trafficFilterManager)
		topologyTasks.SetNodeDeletionRuntime("pc", endpointRuntime)
		topologyTasks.SetNodeDeletionRuntime("switch_l2", endpointRuntime)
		topologyTasks.SetNodeDeletionRuntime("switch_l3", endpointRuntime)
		resourceManager.RegisterInspector("docker", dockerAdapter)
		resourceManager.RegisterInspector("pc", endpointRuntime)
		resourceManager.RegisterInspector("switch_l2", endpointRuntime)
		resourceManager.RegisterInspector("switch_l3", endpointRuntime)
		nodeReconciler := reconcile.NewNodeReconciler(topologyRepository, reconcile.RuntimeDispatch{QEMU: qemuAdapter, Docker: dockerAdapter, Lightweight: endpointRuntime})
		nodeReconciler.SetConcurrency(cfg.StartupConcurrency.QEMU, cfg.StartupConcurrency.Other)
		nodeReconciler.SetResources(resourceManager)
		networkRecovery := reconcile.NewNetworkRecoveryReconciler(topologyRepository, networkService)
		portMappingRecovery := reconcile.NewPortMappingRecoveryReconciler(repositories, portMapper)
		ownershipScanners := []reconcile.OwnershipScanner{
			reconcile.NewQEMUOwnershipScanner(cfg.StateDir),
			reconcile.NewLinuxOwnershipScanner(),
			reconcile.NewOwnedDirectoryScanner("cgroups", cgroupManager.Root, "cgroup", "node"),
			reconcile.NewCaptureStateScanner(cfg.StateDir),
			reconcile.NewOwnedProcessScanner(cfg.StateDir),
			reconcile.NewRuntimeOwnershipSourceScanner("console-proxies", consoleHandlers),
		}
		if networkObjectConsoleHandlers != nil {
			ownershipScanners = append(ownershipScanners, reconcile.NewRuntimeOwnershipSourceScanner("network-object-console-proxies", networkObjectConsoleHandlers))
		}
		if dockerAdapter != nil {
			ownershipScanners = append(ownershipScanners, reconcile.NewRuntimeOwnershipSourceScanner("docker-labels", dockerAdapter))
		}
		ownershipDiscovery := reconcile.NewOwnershipDiscoveryReconciler(repositories, auditService, ownershipScanners...)
		reconcilers := []reconcile.Reconciler{ownershipDiscovery, nodeReconciler}
		if qemuAdapter != nil {
			reconcilers = append(reconcilers, reconcile.NewRuntimeObservationReconciler(repositories, qemuruntime.NewCapabilityProbe(qemuAdapter)))
		}
		var dataPlaneReconciler *reconcile.DataPlaneReconciler
		if dataPlane, dataPlaneErr := linuxnet.NewDataPlane(nil); dataPlaneErr == nil {
			networkService.SetAttachmentRuntime(dataPlane)
			networkService.SetObjectLinkRuntime(dataPlane)
			topologyTasks.SetNodeDeletionLinkRuntime(dataPlane)
			linkReconnectTasks.SetRuntime(reconcile.NewTopologyOperations(dataPlane))
			dataPlaneReconciler = reconcile.NewDataPlaneReconciler(topologyRepository, dataPlane)
			dataPlaneReconciler.SetNetworkObjectReconciler(networkService)
			topologyTasks.SetPostStartReconciler(dataPlaneReconciler)
			reconcilers = append(reconcilers, dataPlaneReconciler)
			reconcilers = append(reconcilers, reconcile.NewLaboratoryDeletionReconciler(topologyRepository, reconcile.RuntimeDispatch{QEMU: qemuAdapter, Docker: dockerAdapter, Lightweight: endpointRuntime}, networkService, dataPlane, captureManager, portMapper, resourceManager, linkRuntime))
		} else {
			logger.Warn("data-plane reconciler unavailable", "error", dataPlaneErr)
		}
		recovery := reconcile.NewStartupRecoveryCoordinator(repositories, reconcile.StartupRecoveryParticipants{
			Nodes:          nodeReconciler,
			NetworkObjects: networkRecovery,
			DurableTasks:   reconcile.NewDurableTaskRecoveryReconciler(taskRunner),
			Reservations:   reconcile.NewTopologyConnectionRecoveryReconciler(repositories),
			DataPlane:      dataPlaneReconciler,
			PortMappings:   portMappingRecovery,
			Captures:       reconcile.NewCaptureRecoveryReconciler(captureManager),
		})
		hostRestarted, bootErr := reconcile.DetectHostRestart(cfg.StateDir)
		if bootErr != nil {
			logger.Warn("host restart detection failed", "error", bootErr)
		}
		if hostRestarted {
			logger.Info("full host restart detected; applying laboratory recovery policies")
			if _, restoreErr := recovery.Execute(ctx, "host_restart", topologyRepository.PrepareHostRecovery); restoreErr != nil {
				logger.Error("host recovery failed", "error", restoreErr)
			}
		} else if _, adoptErr := recovery.Execute(ctx, "service_restart", nil); adoptErr != nil {
			logger.Error("service restart adoption failed", "error", adoptErr)
		}
		workloadReconciler.Start(ctx)
		coordinator := reconcile.NewCoordinator(2*time.Second, logger, reconcilers...)
		coordinator.Start(ctx)
		defer coordinator.Close()
	} else {
		logger.Warn("namespace runtime unavailable", "error", runtimeErr)
	}
	if err = taskRunner.Recover(context.Background()); err != nil {
		logger.Error("recover durable tasks", "error", err)
	}
	go func() {
		if err := server.StartReady(func() error {
			if notifyErr := readiness.Notify("database migrated, recovery reconciled, and authoritative listener bound"); notifyErr != nil {
				return fmt.Errorf("%w: %v", readiness.ErrNotReady, notifyErr)
			}
			return nil
		}); err != nil {
			logger.Error("server stopped", "error", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		logger.Error("graceful shutdown", "error", err)
	}
}

func applyBuildIdentity(cfg *config.Config) {
	if version != "" && version != "dev" {
		cfg.Release.Version = version
	}
	if candidateID != "" && candidateID != "dev" {
		cfg.Release.CandidateID = candidateID
	}
	if binaryDigest != "" {
		cfg.Release.BinaryDigest = binaryDigest
	}
	if contractDigest != "" {
		cfg.Release.ContractDigest = contractDigest
	}
	if builtAt != "" {
		cfg.Release.BuiltAt = builtAt
	}
}

func configuredReleaseIdentity(cfg config.Config) domain.ReleaseIdentity {
	identity := domain.ReleaseIdentity{
		Version:        cfg.Release.Version,
		CandidateID:    cfg.Release.CandidateID,
		BinaryDigest:   cfg.Release.BinaryDigest,
		ContractDigest: cfg.Release.ContractDigest,
	}
	if cfg.Release.BuiltAt != "" {
		if parsed, err := time.Parse(time.RFC3339, cfg.Release.BuiltAt); err == nil {
			identity.BuiltAt = &parsed
		}
	}
	return identity
}
