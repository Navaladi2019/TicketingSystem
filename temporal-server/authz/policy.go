package authz

// The API allow-list of ADR-001 Annex A.2, plus the `nava-operator` row pinned in
// TICK-83 grooming, written method by method from the service descriptors of
// go.temporal.io/api v1.50.1 (WorkflowService, OperatorService) and
// go.temporal.io/server v1.28.4 (AdminService). Wildcards in the ADR are expanded
// here so that a Temporal upgrade that adds a method denies it until someone
// classifies it: policy_test.go fails while any descriptor method is missing from
// both allowList and deniedByDefault, or listed in both.

const (
	wf  = "/temporal.api.workflowservice.v1.WorkflowService/"
	op  = "/temporal.api.operatorservice.v1.OperatorService/"
	adm = "/temporal.server.api.adminservice.v1.AdminService/"
)

// healthCheck is not a Temporal API. It is allowed for every caller so that
// probes work; it reveals only whether the frontend is serving.
const healthCheck = "/grpc.health.v1.Health/Check"

// allowList is every API each identity may call. System is absent: it may call
// everything, in any namespace.
var allowList = map[Identity][]string{
	Relay: {
		wf + "GetSystemInfo",
		wf + "StartWorkflowExecution",
		wf + "SignalWorkflowExecution",
		wf + "SignalWithStartWorkflowExecution",
	},
	Core: {
		wf + "GetSystemInfo",
		wf + "RequestCancelWorkflowExecution",
		wf + "UpdateWorkflowExecution",
		wf + "QueryWorkflow",
		wf + "DescribeWorkflowExecution",
		wf + "GetWorkflowExecutionHistory",
	},
	Worker: {
		wf + "GetSystemInfo",
		wf + "DescribeNamespace",
		wf + "ShutdownWorker",
		wf + "PollWorkflowTaskQueue",
		wf + "PollActivityTaskQueue",
		// RespondWorkflowTask*
		wf + "RespondWorkflowTaskCompleted",
		wf + "RespondWorkflowTaskFailed",
		// RespondActivityTask*, task-token forms only. The *ById forms act on any
		// workflow id without a task token, and M1 has no async completion (T27c).
		wf + "RespondActivityTaskCompleted",
		wf + "RespondActivityTaskFailed",
		wf + "RespondActivityTaskCanceled",
		wf + "RecordActivityTaskHeartbeat",
		wf + "RespondQueryTaskCompleted",
		wf + "GetWorkflowExecutionHistory",
		wf + "RecordWorkerHeartbeat",
	},
	// Reads (Describe*, List*, Count*, Get*) of the three services, and two
	// operational calls named one by one. The Nexus endpoint reads are denied:
	// the grooming row denies "the Nexus endpoint calls" and M1 has no Nexus.
	// The admin Get*ReplicationMessages calls are denied too: the caller's ack
	// position moves replication ack levels, so they are writes (TICK-53 review).
	// The operator-network restriction is T28's; the Authorizer cannot see it.
	Operator: {
		wf + "DescribeNamespace",
		wf + "ListNamespaces",
		wf + "GetWorkflowExecutionHistory",
		wf + "GetWorkflowExecutionHistoryReverse",
		wf + "ListOpenWorkflowExecutions",
		wf + "ListClosedWorkflowExecutions",
		wf + "ListWorkflowExecutions",
		wf + "ListArchivedWorkflowExecutions",
		wf + "CountWorkflowExecutions",
		wf + "GetSearchAttributes",
		wf + "DescribeWorkflowExecution",
		wf + "DescribeTaskQueue",
		wf + "GetClusterInfo",
		wf + "GetSystemInfo",
		wf + "ListTaskQueuePartitions",
		wf + "DescribeSchedule",
		wf + "ListScheduleMatchingTimes",
		wf + "ListSchedules",
		wf + "GetWorkerBuildIdCompatibility",
		wf + "GetWorkerVersioningRules",
		wf + "GetWorkerTaskReachability",
		wf + "DescribeDeployment",
		wf + "DescribeWorkerDeploymentVersion",
		wf + "ListDeployments",
		wf + "GetDeploymentReachability",
		wf + "GetCurrentDeployment",
		wf + "DescribeWorkerDeployment",
		wf + "ListWorkerDeployments",
		wf + "DescribeBatchOperation",
		wf + "ListBatchOperations",
		wf + "DescribeWorkflowRule",
		wf + "ListWorkflowRules",
		wf + "ListWorkers",
		wf + "TerminateWorkflowExecution",

		op + "ListSearchAttributes",
		op + "ListClusters",
		op + "AddSearchAttributes", // T28 registers NavaTenantId

		adm + "DescribeMutableState",
		adm + "DescribeHistoryHost",
		adm + "GetShard",
		adm + "ListHistoryTasks",
		adm + "GetWorkflowExecutionRawHistoryV2",
		adm + "GetWorkflowExecutionRawHistory",
		adm + "GetSearchAttributes",
		adm + "DescribeCluster",
		adm + "ListClusters",
		adm + "ListClusterMembers",
		adm + "GetDLQMessages",
		adm + "GetTaskQueueTasks",
		adm + "GetNamespace",
		adm + "GetDLQTasks",
		adm + "DescribeDLQJob",
		adm + "ListQueues",
		adm + "DescribeTaskQueuePartition",
	},
}

// deniedByDefault is every descriptor method that no identity except System may
// call. It changes nothing at runtime (unlisted means denied); it exists so that
// the completeness test proves each method was classified on purpose.
var deniedByDefault = []string{
	wf + "RegisterNamespace",
	wf + "UpdateNamespace",
	wf + "DeprecateNamespace",
	wf + "ExecuteMultiOperation",
	wf + "RecordActivityTaskHeartbeatById",
	wf + "RespondActivityTaskCompletedById",
	wf + "RespondActivityTaskFailedById",
	wf + "RespondActivityTaskCanceledById",
	wf + "ResetWorkflowExecution",
	wf + "DeleteWorkflowExecution",
	wf + "ScanWorkflowExecutions",
	wf + "ResetStickyTaskQueue",
	wf + "CreateSchedule",
	wf + "UpdateSchedule",
	wf + "PatchSchedule",
	wf + "DeleteSchedule",
	wf + "UpdateWorkerBuildIdCompatibility",
	wf + "UpdateWorkerVersioningRules",
	wf + "SetCurrentDeployment",
	wf + "SetWorkerDeploymentCurrentVersion",
	wf + "DeleteWorkerDeployment",
	wf + "DeleteWorkerDeploymentVersion",
	wf + "SetWorkerDeploymentRampingVersion",
	wf + "UpdateWorkerDeploymentVersionMetadata",
	wf + "PollWorkflowExecutionUpdate",
	wf + "StartBatchOperation",
	wf + "StopBatchOperation",
	wf + "PollNexusTaskQueue",
	wf + "RespondNexusTaskCompleted",
	wf + "RespondNexusTaskFailed",
	wf + "UpdateActivityOptions",
	wf + "UpdateWorkflowExecutionOptions",
	wf + "PauseActivity",
	wf + "UnpauseActivity",
	wf + "ResetActivity",
	wf + "CreateWorkflowRule",
	wf + "DeleteWorkflowRule",
	wf + "TriggerWorkflowRule",

	op + "RemoveSearchAttributes",
	op + "DeleteNamespace",
	op + "AddOrUpdateRemoteCluster",
	op + "RemoveRemoteCluster",
	op + "GetNexusEndpoint",
	op + "CreateNexusEndpoint",
	op + "UpdateNexusEndpoint",
	op + "DeleteNexusEndpoint",
	op + "ListNexusEndpoints",

	adm + "RebuildMutableState",
	adm + "ImportWorkflowExecution",
	adm + "CloseShard",
	adm + "RemoveTask",
	adm + "ReapplyEvents",
	adm + "GetReplicationMessages",
	adm + "GetNamespaceReplicationMessages",
	adm + "GetDLQReplicationMessages",
	adm + "AddSearchAttributes",
	adm + "RemoveSearchAttributes",
	adm + "AddOrUpdateRemoteCluster",
	adm + "RemoveRemoteCluster",
	adm + "PurgeDLQMessages",
	adm + "MergeDLQMessages",
	adm + "RefreshWorkflowTasks",
	adm + "ResendReplicationTasks",
	adm + "DeleteWorkflowExecution",
	adm + "PurgeDLQTasks",
	adm + "MergeDLQTasks",
	adm + "CancelDLQJob",
	adm + "AddTasks",
	adm + "DeepHealthCheck",
	adm + "SyncWorkflowState",
	adm + "GenerateLastHistoryReplicationTasks",
	adm + "ForceUnloadTaskQueuePartition",
	adm + "StreamWorkflowReplicationMessages",
}
