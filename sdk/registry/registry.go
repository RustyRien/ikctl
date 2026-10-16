// Package registry describes the InfraKitchen entity kinds and how they map to
// GraphQL operations. Every entity follows the same conventions:
//
//	x(id)                     get one
//	xs(filter, sort, range)   list
//	xsCount(filter)           count
//	xActions(id)              allowed actions
//	createX(input)            create
//	updateX(id, input)        update
//	xAction(id, input)        run an action
//	deleteX(id)               delete
//
// Adding a kind means adding one entry to the kinds slice.
package registry

import (
	"slices"
	"strings"
)

// Verb is an operation supported by a kind.
type Verb string

// Supported verbs.
const (
	VerbGet    Verb = "get"
	VerbList   Verb = "list"
	VerbCreate Verb = "create"
	VerbUpdate Verb = "update"
	VerbDelete Verb = "delete"
	VerbAction Verb = "action"
)

var (
	readVerbs = []Verb{VerbGet, VerbList}
	allVerbs  = []Verb{VerbGet, VerbList, VerbCreate, VerbUpdate, VerbDelete, VerbAction}
)

// ColumnType controls how a table column value is rendered.
type ColumnType string

// Column types.
const (
	ColumnText ColumnType = ""
	ColumnAge  ColumnType = "age"
)

// Column is a table column. Path is a dot-separated path into the object (e.g. "template.name").
type Column struct {
	Header string     `json:"header"`
	Path   string     `json:"path"`
	Type   ColumnType `json:"type,omitempty"`
	Wide   bool       `json:"wide,omitempty"` // only shown with -o wide
}

// Kind describes one entity kind.
type Kind struct {
	// Name is the CLI kind name, e.g. "resource" or "source-code-version".
	Name string `json:"name"`
	// Plural is the plural CLI name, e.g. "resources".
	Plural string `json:"plural"`
	// Aliases are short names, e.g. "rs".
	Aliases []string `json:"aliases,omitempty"`
	// Title is the type title used in list envelopes, e.g. "Resource".
	Title string `json:"title"`
	// GraphQLName is the singular camelCase GraphQL name, e.g. "sourceCodeVersion".
	GraphQLName string `json:"graphqlName"`
	// TypeName is the GraphQL object type, e.g. "ResourceType".
	TypeName string `json:"typeName"`
	// IDType is the GraphQL type of the id argument.
	IDType string `json:"idType"`
	// NameFilter is the snake_case filter key used to look objects up by name
	// ("" if the kind can only be addressed by id).
	NameFilter string `json:"nameFilter,omitempty"`
	// NamePath is the object field shown as the object's name.
	NamePath string `json:"namePath,omitempty"`
	// UpdateArg is the name of the update mutation input argument ("input" by default).
	UpdateArg string `json:"-"`
	// ListFields is the default GraphQL selection for lists.
	ListFields string `json:"-"`
	// DescribeFields is the GraphQL selection for a single object.
	DescribeFields string `json:"-"`
	// Columns are the table columns.
	Columns []Column `json:"columns"`
	// Verbs are the supported operations.
	Verbs []Verb `json:"verbs"`
}

// Supports reports whether the kind supports v.
func (k Kind) Supports(v Verb) bool { return slices.Contains(k.Verbs, v) }

// HasActionsQuery reports whether the server exposes an xActions(id) query for the kind.
func (k Kind) HasActionsQuery() bool {
	return k.Supports(VerbAction) || k.Supports(VerbUpdate) || k.Supports(VerbDelete)
}

// HasNames reports whether objects of this kind can be looked up by name.
func (k Kind) HasNames() bool { return k.NameFilter != "" }

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// GetOp returns the get-one query name.
func (k Kind) GetOp() string { return k.GraphQLName }

// ListOp returns the list query name.
func (k Kind) ListOp() string { return k.GraphQLName + "s" }

// CountOp returns the count query name.
func (k Kind) CountOp() string { return k.GraphQLName + "sCount" }

// ActionsOp returns the allowed-actions query name.
func (k Kind) ActionsOp() string { return k.GraphQLName + "Actions" }

// CreateOp returns the create mutation name.
func (k Kind) CreateOp() string { return "create" + upperFirst(k.GraphQLName) }

// UpdateOp returns the update mutation name.
func (k Kind) UpdateOp() string { return "update" + upperFirst(k.GraphQLName) }

// ActionOp returns the action mutation name.
func (k Kind) ActionOp() string { return k.GraphQLName + "Action" }

// DeleteOp returns the delete mutation name.
func (k Kind) DeleteOp() string { return "delete" + upperFirst(k.GraphQLName) }

// CreateInputType returns the GraphQL input type of the create mutation.
func (k Kind) CreateInputType() string { return k.Title + "CreateInput" }

// UpdateInputType returns the GraphQL input type of the update mutation.
func (k Kind) UpdateInputType() string { return k.Title + "UpdateInput" }

// ActionInputType returns the GraphQL input type of the action mutation.
func (k Kind) ActionInputType() string { return k.Title + "ActionInput" }

// UpdateArgName returns the update mutation input argument name.
func (k Kind) UpdateArgName() string {
	if k.UpdateArg == "" {
		return "input"
	}
	return k.UpdateArg
}

// ListTitle returns the envelope kind for lists, e.g. "ResourceList".
func (k Kind) ListTitle() string { return k.Title + "List" }

// All returns all known kinds, sorted by name.
func All() []Kind {
	out := slices.Clone(kinds)
	slices.SortFunc(out, func(a, b Kind) int { return strings.Compare(a.Name, b.Name) })
	return out
}

// Lookup finds a kind by name, plural, alias, title or GraphQL name (case-insensitive).
func Lookup(s string) (Kind, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	norm := strings.NewReplacer("-", "", "_", "").Replace(s)
	for _, k := range kinds {
		candidates := append([]string{k.Name, k.Plural, k.Title, k.GraphQLName, k.ListOp()}, k.Aliases...)
		for _, c := range candidates {
			c = strings.ToLower(c)
			if s == c || norm == strings.NewReplacer("-", "", "_", "").Replace(c) {
				return k, true
			}
		}
	}
	return Kind{}, false
}

// Common selections.
const (
	userRef     = "{ id identifier displayName }"
	auditFields = "createdAt updatedAt"
)

var nameStatusColumns = []Column{
	{Header: "NAME", Path: "name"},
	{Header: "STATUS", Path: "status"},
	{Header: "AGE", Path: "createdAt", Type: ColumnAge},
	{Header: "ID", Path: "id", Wide: true},
	{Header: "LABELS", Path: "labels", Wide: true},
}

var kinds = []Kind{
	{
		Name: "resource", Plural: "resources", Aliases: []string{"rs", "res"}, Title: "Resource",
		GraphQLName: "resource", TypeName: "ResourceType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields: "id name state status labels abstract template { id name } " + auditFields,
		DescribeFields: "id name description state status abstract labels storagePath variables outputs " +
			"dependencyTags dependencyConfig revisionNumber " + auditFields + " " +
			"template { id name template } sourceCodeVersion { id identifier } storage { id name } " +
			"tool { id name version } workspace { id name } project { id name } " +
			"integrationIds { id name } secretIds { id name } parents { id name } children { id name } " +
			"creator " + userRef,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "TEMPLATE", Path: "template.name"},
			{Header: "STATE", Path: "state"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
			{Header: "LABELS", Path: "labels", Wide: true},
		},
		Verbs: allVerbs,
	},
	{
		Name: "template", Plural: "templates", Aliases: []string{"tpl", "tmpl"}, Title: "Template",
		GraphQLName: "template", TypeName: "TemplateType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields: "id name template status labels abstract cloudResourceTypes " + auditFields,
		DescribeFields: "id name description documentation template cloudResourceTypes abstract configuration " +
			"labels status revisionNumber resourcesCount sourceCodeVersionsCount " + auditFields + " " +
			"parents { id name } children { id name } creator " + userRef,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "SLUG", Path: "template"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
			{Header: "LABELS", Path: "labels", Wide: true},
		},
		Verbs: allVerbs,
	},
	{
		Name: "integration", Plural: "integrations", Aliases: []string{"int"}, Title: "Integration",
		GraphQLName: "integration", TypeName: "IntegrationType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields: "id name integrationType integrationProvider status labels " + auditFields,
		DescribeFields: "id name description integrationType integrationProvider configuration labels status " +
			"revisionNumber resourceCount sourceCodeCount workspaceCount storageCount executorCount " +
			auditFields + " creator " + userRef,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "TYPE", Path: "integrationType"},
			{Header: "PROVIDER", Path: "integrationProvider"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
			{Header: "LABELS", Path: "labels", Wide: true},
		},
		Verbs: allVerbs,
	},
	{
		Name: "source-code", Plural: "source-codes", Aliases: []string{"sc", "sourcecode"}, Title: "SourceCode",
		GraphQLName: "sourceCode", TypeName: "SourceCodeType", IDType: "UUID!",
		NameFilter: "source_code_url", NamePath: "identifier",
		ListFields: "id identifier sourceCodeUrl sourceCodeProvider sourceCodeLanguage defaultBranch status labels " + auditFields,
		DescribeFields: "id identifier description sourceCodeUrl sourceCodeProvider sourceCodeLanguage defaultBranch " +
			"gitTags gitBranches labels status revisionNumber sourceCodeVersionCount commitCount " + auditFields + " " +
			"integration { id name } creator " + userRef,
		Columns: []Column{
			{Header: "URL", Path: "sourceCodeUrl"},
			{Header: "PROVIDER", Path: "sourceCodeProvider"},
			{Header: "LANGUAGE", Path: "sourceCodeLanguage"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
			{Header: "DEFAULT BRANCH", Path: "defaultBranch", Wide: true},
		},
		Verbs: allVerbs,
	},
	{
		Name: "source-code-version", Plural: "source-code-versions", Aliases: []string{"scv"}, Title: "SourceCodeVersion",
		GraphQLName: "sourceCodeVersion", TypeName: "SourceCodeVersionType", IDType: "UUID!",
		NameFilter: "source_code_version", NamePath: "identifier",
		ListFields: "id identifier sourceCodeVersion sourceCodeBranch sourceCodeFolder lifecycleState status labels " +
			auditFields + " template { id name }",
		DescribeFields: "id identifier description sourceCodeVersion sourceCodeBranch sourceCodeFolder lifecycleState " +
			"breakingChanges index status labels variables outputs revisionNumber resourcesCount " + auditFields + " " +
			"template { id name } sourceCode { id identifier } creator " + userRef,
		Columns: []Column{
			{Header: "IDENTIFIER", Path: "identifier"},
			{Header: "TEMPLATE", Path: "template.name"},
			{Header: "LIFECYCLE", Path: "lifecycleState"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
		},
		Verbs: allVerbs,
	},
	{
		Name: "workspace", Plural: "workspaces", Aliases: []string{"ws"}, Title: "Workspace",
		GraphQLName: "workspace", TypeName: "WorkspaceType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields: "id name workspaceProvider status labels " + auditFields,
		DescribeFields: "id name description workspaceProvider configuration status labels resourcesCount " +
			auditFields + " integration { id name } creator " + userRef,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "PROVIDER", Path: "workspaceProvider"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
		},
		Verbs: []Verb{VerbGet, VerbList, VerbCreate, VerbUpdate, VerbDelete},
	},
	{
		Name: "storage", Plural: "storages", Aliases: []string{"st"}, Title: "Storage",
		GraphQLName: "storage", TypeName: "StorageType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields: "id name storageType storageProvider state status labels " + auditFields,
		DescribeFields: "id name description storageType storageProvider configuration state status labels " +
			"revisionNumber resourcesCount executorsCount " + auditFields + " integration { id name } creator " + userRef,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "PROVIDER", Path: "storageProvider"},
			{Header: "STATE", Path: "state"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
		},
		Verbs: allVerbs,
	},
	{
		Name: "secret", Plural: "secrets", Aliases: []string{"sec"}, Title: "Secret",
		GraphQLName: "secret", TypeName: "SecretType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields: "id name secretType secretProvider state status labels " + auditFields,
		DescribeFields: "id name description secretType secretProvider configuration state status labels " +
			"revisionNumber resourcesCount executorsCount " + auditFields + " integration { id name } creator " + userRef,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "TYPE", Path: "secretType"},
			{Header: "PROVIDER", Path: "secretProvider"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
		},
		Verbs: allVerbs,
	},
	{
		Name: "executor", Plural: "executors", Aliases: []string{"exec", "ex"}, Title: "Executor",
		GraphQLName: "executor", TypeName: "ExecutorType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields: "id name runtime state status labels " + auditFields,
		DescribeFields: "id name description runtime commandArgs sourceCodeVersion sourceCodeBranch sourceCodeFolder " +
			"storagePath state status labels revisionNumber " + auditFields + " " +
			"sourceCode { id identifier } storage { id name } tool { id name version } " +
			"integrationIds { id name } secretIds { id name } creator " + userRef,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "RUNTIME", Path: "runtime"},
			{Header: "STATE", Path: "state"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
		},
		Verbs: allVerbs,
	},
	{
		Name: "project", Plural: "projects", Aliases: []string{"proj", "prj"}, Title: "Project",
		GraphQLName: "project", TypeName: "ProjectType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields: "id name status labels resourcesCount servicesCount " + auditFields,
		DescribeFields: "id name description workspaceId configuration dependencyTags dependencyConfig labels status " +
			"revisionNumber resourcesCount servicesCount " + auditFields + " owners " + userRef + " creator " + userRef,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "STATUS", Path: "status"},
			{Header: "RESOURCES", Path: "resourcesCount"},
			{Header: "SERVICES", Path: "servicesCount"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
		},
		Verbs: allVerbs,
	},
	{
		Name: "service", Plural: "services", Aliases: []string{"svc"}, Title: "Service",
		GraphQLName: "service", TypeName: "ServiceType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields: "id name displayName repositoryUrl labels " + auditFields + " project { id name }",
		DescribeFields: "id name displayName description projectId repositoryUrl labels revisionNumber " + auditFields +
			" project { id name } owners " + userRef + " creator " + userRef,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "PROJECT", Path: "project.name"},
			{Header: "REPOSITORY", Path: "repositoryUrl"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
		},
		Verbs: []Verb{VerbGet, VerbList, VerbCreate, VerbUpdate, VerbDelete},
	},
	{
		Name: "blueprint", Plural: "blueprints", Aliases: []string{"bp"}, Title: "Blueprint",
		GraphQLName: "blueprint", TypeName: "BlueprintType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields: "id name status labels " + auditFields,
		DescribeFields: "id name description wiring defaultVariables configuration labels status revisionNumber " +
			auditFields + " templates { id name } externalTemplates { id name } workflows { id action status } " +
			"creator " + userRef,
		Columns: nameStatusColumns,
		Verbs:   allVerbs,
	},
	{
		Name: "workflow", Plural: "workflows", Aliases: []string{"wf"}, Title: "Workflow",
		GraphQLName: "workflow", TypeName: "WorkflowType", IDType: "UUID!",
		ListFields: "id action status errorMessage startedAt completedAt createdAt",
		DescribeFields: "id action status errorMessage wiringSnapshot startedAt completedAt createdAt " +
			"steps { id position status errorMessage startedAt completedAt template { id name } resource { id name } } " +
			"creator " + userRef,
		Columns: []Column{
			{Header: "ID", Path: "id"},
			{Header: "ACTION", Path: "action"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ERROR", Path: "errorMessage", Wide: true},
		},
		Verbs: []Verb{VerbGet, VerbList, VerbUpdate, VerbDelete, VerbAction},
	},
	{
		Name: "tool", Plural: "tools", Title: "Tool",
		GraphQLName: "tool", TypeName: "ToolType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields: "id name version os arch status isDefault " + auditFields,
		DescribeFields: "id name version os arch executable sourceUrl sha256 size status errorMessage isDefault " +
			"resourcesCount executorsCount " + auditFields + " creator " + userRef,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "VERSION", Path: "version"},
			{Header: "DEFAULT", Path: "isDefault"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
			{Header: "OS/ARCH", Path: "os", Wide: true},
		},
		Verbs: []Verb{VerbGet, VerbList, VerbDelete, VerbAction},
	},
	{
		Name: "task", Plural: "tasks", Title: "Task",
		GraphQLName: "task", TypeName: "TaskType", IDType: "UUID!",
		ListFields:     "id action entity entityId state status runAt cron error " + auditFields,
		DescribeFields: "id action entity entityId entityData state status runAt cron timezone error " + auditFields + " creator " + userRef,
		Columns: []Column{
			{Header: "ID", Path: "id"},
			{Header: "ENTITY", Path: "entity"},
			{Header: "ENTITY ID", Path: "entityId"},
			{Header: "STATE", Path: "state"},
			{Header: "STATUS", Path: "status"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ERROR", Path: "error", Wide: true},
		},
		Verbs: readVerbs,
	},
	{
		Name: "audit-log", Plural: "audit-logs", Aliases: []string{"al", "auditlog"}, Title: "AuditLog",
		GraphQLName: "auditLog", TypeName: "AuditLogType", IDType: "UUID!",
		ListFields:     "id action model entityId revisionNumber createdAt creator " + userRef,
		DescribeFields: "id userId action model entityId revisionNumber entityData metadata createdAt creator " + userRef,
		Columns: []Column{
			{Header: "ID", Path: "id"},
			{Header: "ACTION", Path: "action"},
			{Header: "MODEL", Path: "model"},
			{Header: "ENTITY ID", Path: "entityId"},
			{Header: "USER", Path: "creator.identifier"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
		},
		Verbs: readVerbs,
	},
	{
		Name: "worker", Plural: "workers", Title: "Worker",
		GraphQLName: "worker", TypeName: "WorkerType", IDType: "UUID!",
		NameFilter: "name", NamePath: "name",
		ListFields:     "id name host status currentTask tasksCompleted " + auditFields,
		DescribeFields: "id name host hostMetadata status currentTask tasksCompleted " + auditFields,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "HOST", Path: "host"},
			{Header: "STATUS", Path: "status"},
			{Header: "COMPLETED", Path: "tasksCompleted"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
		},
		Verbs: readVerbs,
	},
	{
		Name: "user", Plural: "users", Aliases: []string{"usr"}, Title: "User",
		GraphQLName: "user", TypeName: "UserType", IDType: "UUID!",
		NameFilter: "identifier", NamePath: "identifier",
		ListFields:     "id identifier email displayName provider deactivated " + auditFields,
		DescribeFields: "id identifier email firstName lastName displayName provider deactivated description isPrimary " + auditFields,
		Columns: []Column{
			{Header: "IDENTIFIER", Path: "identifier"},
			{Header: "EMAIL", Path: "email"},
			{Header: "PROVIDER", Path: "provider"},
			{Header: "DEACTIVATED", Path: "deactivated"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
			{Header: "ID", Path: "id", Wide: true},
		},
		Verbs: readVerbs,
	},
	{
		Name: "permission", Plural: "permissions", Aliases: []string{"perm"}, Title: "Permission",
		GraphQLName: "permission", TypeName: "PermissionType", IDType: "UUID!",
		ListFields:     "id ptype v0 v1 v2 v3 " + auditFields,
		DescribeFields: "id ptype v0 v1 v2 v3 v4 v5 entityData " + auditFields + " userData " + userRef + " creator " + userRef,
		Columns: []Column{
			{Header: "ID", Path: "id"},
			{Header: "PTYPE", Path: "ptype"},
			{Header: "SUBJECT", Path: "v0"},
			{Header: "OBJECT", Path: "v1"},
			{Header: "ACTION", Path: "v2"},
			{Header: "AGE", Path: "createdAt", Type: ColumnAge},
		},
		Verbs: []Verb{VerbGet, VerbList, VerbDelete},
	},
}
