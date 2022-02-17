// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package graphql

import (
	"fmt"
	"io"
	"strconv"
)

// Action describes user intention to resolve & execute a given Interface or Implementation.
type Action struct {
	Name      string        `json:"name"`
	CreatedAt Timestamp     `json:"createdAt"`
	Input     *ActionInput  `json:"input"`
	Output    *ActionOutput `json:"output"`
	// Contains reference to the Implementation or Interface manifest
	ActionRef *ManifestReference `json:"actionRef"`
	// Indicates if user approved this Action to run
	Run bool `json:"run"`
	// Indicates if user canceled the workflow. CURRENTLY NOT SUPPORTED.
	Cancel bool `json:"cancel"`
	// Specifies whether the Action performs server-side test without actually running the Action.
	// For now it only lints the rendered Argo manifests and does not execute any workflow.
	DryRun         bool        `json:"dryRun"`
	RenderedAction interface{} `json:"renderedAction"`
	// CURRENTLY NOT IMPLEMENTED.
	RenderingAdvancedMode *ActionRenderingAdvancedMode `json:"renderingAdvancedMode"`
	// CURRENTLY NOT IMPLEMENTED.
	RenderedActionOverride interface{}   `json:"renderedActionOverride"`
	Status                 *ActionStatus `json:"status"`
}

// Client input of Action details, that are used for create and update Action operations (PUT-like operation)
type ActionDetailsInput struct {
	Name  string           `json:"name"`
	Input *ActionInputData `json:"input"`
	// Contains reference to the Implementation or Interface manifest
	ActionRef *ManifestReferenceInput `json:"actionRef"`
	// Specifies whether the Action performs server-side test without actually running the Action
	// For now it only lints the rendered Argo manifests and does not execute any workflow.
	DryRun *bool `json:"dryRun"`
	// Enables advanced rendering mode for Action. CURRENTLY NOT IMPLEMENTED.
	AdvancedRendering *bool `json:"advancedRendering"`
	// Used to override the rendered action. CURRENTLY NOT IMPLEMENTED.
	RenderedActionOverride *JSON `json:"renderedActionOverride"`
}

// Set of filters for Action list
type ActionFilter struct {
	Phase        *ActionStatusPhase      `json:"phase"`
	NameRegex    *string                 `json:"nameRegex"`
	InterfaceRef *ManifestReferenceInput `json:"interfaceRef"`
}

// Describes input of an Action
type ActionInput struct {
	// Validated against JSON schema from Interface
	Parameters    interface{}                 `json:"parameters"`
	TypeInstances []*InputTypeInstanceDetails `json:"typeInstances"`
	// Contains the one-time Action policy, which is merged with other Capact policies
	ActionPolicy *Policy `json:"actionPolicy"`
}

// Client input that modifies input of a given Action
type ActionInputData struct {
	// During rendering, it is validated against JSON schema from Interface of the resolved action
	Parameters *JSON `json:"parameters"`
	// Required and optional TypeInstances for Action
	TypeInstances []*InputTypeInstanceData `json:"typeInstances"`
	// Contains the optional one-time Action policy, which is merged with other Capact policies
	ActionPolicy *PolicyInput `json:"actionPolicy"`
}

// Describes output of an Action
type ActionOutput struct {
	TypeInstances []*OutputTypeInstanceDetails `json:"typeInstances"`
}

// Properties related to Action advanced rendering. CURRENTLY NOT IMPLEMENTED.
type ActionRenderingAdvancedMode struct {
	Enabled bool `json:"enabled"`
	// Optional TypeInstances for current rendering iteration
	TypeInstancesForRenderingIteration []*InputTypeInstanceToProvide `json:"typeInstancesForRenderingIteration"`
}

// Status of the Action
type ActionStatus struct {
	Phase     ActionStatusPhase `json:"phase"`
	Timestamp Timestamp         `json:"timestamp"`
	Message   *string           `json:"message"`
	Runner    *RunnerStatus     `json:"runner"`
	// CURRENTLY NOT IMPLEMENTED.
	CreatedBy *UserInfo `json:"createdBy"`
	// CURRENTLY NOT IMPLEMENTED.
	RunBy *UserInfo `json:"runBy"`
	// CURRENTLY NOT IMPLEMENTED.
	CanceledBy *UserInfo `json:"canceledBy"`
}

type AdditionalParameter struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type AdditionalParameterInput struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type AdditionalTypeInstanceReferenceInput struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// Input used for continuing Action rendering in advanced mode
type AdvancedModeContinueRenderingInput struct {
	// Optional TypeInstances for a given rendering iteration
	TypeInstances []*InputTypeInstanceData `json:"typeInstances"`
}

// Client input for Input TypeInstance
type InputTypeInstanceData struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// Describes input TypeInstance of an Action
type InputTypeInstanceDetails struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Describes optional input TypeInstance of advanced rendering iteration
type InputTypeInstanceToProvide struct {
	Name    string             `json:"name"`
	TypeRef *ManifestReference `json:"typeRef"`
}

type InterfacePolicy struct {
	Rules []*RulesForInterface `json:"rules"`
}

type InterfacePolicyInput struct {
	Rules []*RulesForInterfaceInput `json:"rules"`
}

type ManifestReference struct {
	Path     string `json:"path"`
	Revision string `json:"revision"`
}

type ManifestReferenceInput struct {
	// Full path for the manifest
	Path string `json:"path"`
	// If not provided, latest revision for a given manifest is used
	Revision *string `json:"revision"`
}

// Describes output TypeInstance of an Action
type OutputTypeInstanceDetails struct {
	ID      string                      `json:"id"`
	TypeRef *ManifestReference          `json:"typeRef"`
	Backend *TypeInstanceBackendDetails `json:"backend"`
}

type Policy struct {
	Interface    *InterfacePolicy    `json:"interface"`
	TypeInstance *TypeInstancePolicy `json:"typeInstance"`
}

type PolicyInput struct {
	Interface    *InterfacePolicyInput    `json:"interface"`
	TypeInstance *TypeInstancePolicyInput `json:"typeInstance"`
}

type PolicyRuleImplementationConstraintsInput struct {
	// Refers a specific required TypeInstance by path and optional revision.
	Requires []*ManifestReferenceInput `json:"requires"`
	// Refers a specific Attribute by path and optional revision.
	Attributes []*ManifestReferenceInput `json:"attributes"`
	// Refers a specific Implementation with exact path.
	Path *string `json:"path"`
}

type PolicyRuleInjectDataInput struct {
	RequiredTypeInstances   []*RequiredTypeInstanceReferenceInput   `json:"requiredTypeInstances"`
	AdditionalParameters    []*AdditionalParameterInput             `json:"additionalParameters"`
	AdditionalTypeInstances []*AdditionalTypeInstanceReferenceInput `json:"additionalTypeInstances"`
}

type PolicyRuleInput struct {
	ImplementationConstraints *PolicyRuleImplementationConstraintsInput `json:"implementationConstraints"`
	Inject                    *PolicyRuleInjectDataInput                `json:"inject"`
}

type RequiredTypeInstanceReferenceInput struct {
	ID          string  `json:"id"`
	Description *string `json:"description"`
}

type RulesForInterface struct {
	Interface *ManifestReferenceWithOptionalRevision `json:"interface"`
	OneOf     []*PolicyRule                          `json:"oneOf"`
}

type RulesForInterfaceInput struct {
	Interface *ManifestReferenceInput `json:"interface"`
	OneOf     []*PolicyRuleInput      `json:"oneOf"`
}

type RulesForTypeInstance struct {
	TypeRef *ManifestReferenceWithOptionalRevision `json:"typeRef"`
	Backend *TypeInstanceBackendRule               `json:"backend"`
}

type RulesForTypeInstanceInput struct {
	TypeRef *ManifestReferenceInput       `json:"typeRef"`
	Backend *TypeInstanceBackendRuleInput `json:"backend"`
}

// Additional Action status from the Runner
type RunnerStatus struct {
	// Status of a given Runner e.g. Argo Workflow Runner status object with argoWorkflowRef field
	Status interface{} `json:"status"`
}

type TypeInstanceBackendDetails struct {
	ID       string `json:"id"`
	Abstract bool   `json:"abstract"`
}

type TypeInstanceBackendRule struct {
	ID          string  `json:"id"`
	Description *string `json:"description"`
}

type TypeInstanceBackendRuleInput struct {
	ID          string  `json:"id"`
	Description *string `json:"description"`
}

type TypeInstancePolicy struct {
	Rules []*RulesForTypeInstance `json:"rules"`
}

type TypeInstancePolicyInput struct {
	Rules []*RulesForTypeInstanceInput `json:"rules"`
}

// Stores user information
type UserInfo struct {
	Username string      `json:"username"`
	Groups   []string    `json:"groups"`
	Extra    interface{} `json:"extra"`
}

// Current phase of the Action
type ActionStatusPhase string

const (
	ActionStatusPhaseInitial                        ActionStatusPhase = "INITIAL"
	ActionStatusPhaseBeingRendered                  ActionStatusPhase = "BEING_RENDERED"
	ActionStatusPhaseAdvancedModeRenderingIteration ActionStatusPhase = "ADVANCED_MODE_RENDERING_ITERATION"
	ActionStatusPhaseReadyToRun                     ActionStatusPhase = "READY_TO_RUN"
	ActionStatusPhaseRunning                        ActionStatusPhase = "RUNNING"
	ActionStatusPhaseBeingCanceled                  ActionStatusPhase = "BEING_CANCELED"
	ActionStatusPhaseCanceled                       ActionStatusPhase = "CANCELED"
	ActionStatusPhaseSucceeded                      ActionStatusPhase = "SUCCEEDED"
	ActionStatusPhaseFailed                         ActionStatusPhase = "FAILED"
)

var AllActionStatusPhase = []ActionStatusPhase{
	ActionStatusPhaseInitial,
	ActionStatusPhaseBeingRendered,
	ActionStatusPhaseAdvancedModeRenderingIteration,
	ActionStatusPhaseReadyToRun,
	ActionStatusPhaseRunning,
	ActionStatusPhaseBeingCanceled,
	ActionStatusPhaseCanceled,
	ActionStatusPhaseSucceeded,
	ActionStatusPhaseFailed,
}

func (e ActionStatusPhase) IsValid() bool {
	switch e {
	case ActionStatusPhaseInitial, ActionStatusPhaseBeingRendered, ActionStatusPhaseAdvancedModeRenderingIteration, ActionStatusPhaseReadyToRun, ActionStatusPhaseRunning, ActionStatusPhaseBeingCanceled, ActionStatusPhaseCanceled, ActionStatusPhaseSucceeded, ActionStatusPhaseFailed:
		return true
	}
	return false
}

func (e ActionStatusPhase) String() string {
	return string(e)
}

func (e *ActionStatusPhase) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = ActionStatusPhase(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid ActionStatusPhase", str)
	}
	return nil
}

func (e ActionStatusPhase) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
