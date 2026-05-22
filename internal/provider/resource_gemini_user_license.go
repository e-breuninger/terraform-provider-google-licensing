package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &geminiUserLicenseResource{}
	_ resource.ResourceWithConfigure = &geminiUserLicenseResource{}
)

func NewGeminiUserLicenseResource() resource.Resource {
	return &geminiUserLicenseResource{}
}

type geminiUserLicenseResource struct {
	httpClient *http.Client
}

type geminiUserLicenseModel struct {
	ID            types.String `tfsdk:"id"`
	Project       types.String `tfsdk:"project"`
	Location      types.String `tfsdk:"location"`
	UserID        types.String `tfsdk:"user_id"`
	LicenseConfig types.String `tfsdk:"license_config"`
	State         types.String `tfsdk:"state"`
}

func (r *geminiUserLicenseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gemini_user_license"
}

func (r *geminiUserLicenseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Assigns a Gemini Enterprise / NotebookLM Enterprise license to a user via the
[Discovery Engine API](https://cloud.google.com/generative-ai-app-builder/docs/reference/rest).

## Import

Existing assignments can be imported using the composite ID:

` + "```" + `shell
terraform import googleenterpriselicense_gemini_user_license.example <project>/<location>/<user_id>
` + "```",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the form `<project>/<location>/<user_id>`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The GCP project ID that holds the Gemini Enterprise subscription.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Multi-region for the license: `global`, `us`, or `eu`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The user's primary email address.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"license_config": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Full resource path of the license configuration, e.g. " +
					"`projects/PROJECT_NUMBER/locations/LOCATION/licenseConfigs/SUBSCRIPTION_ID`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"state": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "License assignment state returned by the API (e.g. `LICENSE_ASSIGNMENT_STATE_ASSIGNED`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *geminiUserLicenseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pc, ok := req.ProviderData.(*providerClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *providerClient, got %T", req.ProviderData),
		)
		return
	}
	r.httpClient = pc.HTTPClient
}

func (r *geminiUserLicenseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan geminiUserLicenseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := plan.Project.ValueString()
	location := plan.Location.ValueString()
	userID := plan.UserID.ValueString()
	licenseConfig := plan.LicenseConfig.ValueString()

	tflog.Debug(ctx, "Assigning Gemini Enterprise user license", map[string]any{
		"project":        project,
		"location":       location,
		"user_id":        userID,
		"license_config": licenseConfig,
	})

	body := newBatchUpdateRequest(userID, licenseConfig, false)
	if reqJSON, err := json.Marshal(body); err == nil {
		tflog.Debug(ctx, "batchUpdateUserLicenses request", map[string]any{"body": string(reqJSON)})
	}
	respBody, err := r.batchUpdate(ctx, project, location, body)
	if err != nil {
		resp.Diagnostics.AddError("Error assigning Gemini Enterprise license", err.Error())
		return
	}
	tflog.Debug(ctx, "batchUpdateUserLicenses response", map[string]any{"body": string(respBody)})

	plan.ID = types.StringValue(geminiLicenseID(project, location, userID))
	plan.State = types.StringValue("ASSIGNED")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *geminiUserLicenseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state geminiUserLicenseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := state.Project.ValueString()
	location := state.Location.ValueString()
	userID := state.UserID.ValueString()
	expectedLicenseConfig := state.LicenseConfig.ValueString()

	license, found, err := r.findUserLicense(ctx, project, location, userID, expectedLicenseConfig)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Gemini Enterprise license", err.Error())
		return
	}

	if !found || license.LicenseConfig == "" ||
		license.LicenseAssignmentState == "UNASSIGNED" || license.LicenseAssignmentState == "NO_LICENSE" {
		tflog.Warn(ctx, "Gemini user license not found, removing from state",
			map[string]any{"id": state.ID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	state.LicenseConfig = types.StringValue(license.LicenseConfig)
	state.State = types.StringValue(license.LicenseAssignmentState)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *geminiUserLicenseResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "All attribute changes require resource replacement.")
}

func (r *geminiUserLicenseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state geminiUserLicenseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := state.Project.ValueString()
	location := state.Location.ValueString()
	userID := state.UserID.ValueString()

	tflog.Debug(ctx, "Revoking Gemini Enterprise user license", map[string]any{
		"project":  project,
		"location": location,
		"user_id":  userID,
	})

	body := newBatchUpdateRequest(userID, "", true)
	if _, err := r.batchUpdate(ctx, project, location, body); err != nil {
		resp.Diagnostics.AddError("Error revoking Gemini Enterprise license", err.Error())
	}
}

type geminiAPIError struct {
	StatusCode int
	Body       []byte
}

func (e *geminiAPIError) Error() string {
	return fmt.Sprintf("Gemini API error (status %d): %s", e.StatusCode, string(e.Body))
}

func isGeminiNotFound(err error) bool {
	if e, ok := err.(*geminiAPIError); ok {
		return e.StatusCode == 404
	}
	return false
}

func (r *geminiUserLicenseResource) userStoreBase(project, location string) string {
	return fmt.Sprintf(
		"https://%s-discoveryengine.googleapis.com/v1/projects/%s/locations/%s/userStores/default_user_store",
		location, project, location,
	)
}

func (r *geminiUserLicenseResource) doRequest(ctx context.Context, method, url string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("building HTTP request: %w", err)
	}
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	httpResp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, &geminiAPIError{StatusCode: httpResp.StatusCode, Body: respBody}
	}
	return respBody, nil
}

type batchUpdateRequest struct {
	InlineSource                 batchInlineSource `json:"inlineSource"`
	DeleteUnassignedUserLicenses bool              `json:"deleteUnassignedUserLicenses"`
}

type batchInlineSource struct {
	UserLicenses []geminiUserLicenseEntry `json:"userLicenses"`
	UpdateMask   string                   `json:"updateMask"`
}

type geminiUserLicenseEntry struct {
	UserPrincipal          string `json:"userPrincipal"`
	LicenseConfig          string `json:"licenseConfig,omitempty"`
	LicenseAssignmentState string `json:"licenseAssignmentState,omitempty"`
}

type listUserLicensesResponse struct {
	UserLicenses  []geminiUserLicenseEntry `json:"userLicenses"`
	NextPageToken string                   `json:"nextPageToken"`
}

func newBatchUpdateRequest(userID, licenseConfig string, deleteUnassigned bool) batchUpdateRequest {
	entry := geminiUserLicenseEntry{UserPrincipal: userID, LicenseConfig: licenseConfig}
	var req batchUpdateRequest
	req.InlineSource.UserLicenses = []geminiUserLicenseEntry{entry}
	req.InlineSource.UpdateMask = "licenseConfig"
	req.DeleteUnassignedUserLicenses = deleteUnassigned
	return req
}

func (r *geminiUserLicenseResource) batchUpdate(ctx context.Context, project, location string, body batchUpdateRequest) ([]byte, error) {
	url := r.userStoreBase(project, location) + ":batchUpdateUserLicenses"
	return r.doRequest(ctx, http.MethodPost, url, body)
}

func (r *geminiUserLicenseResource) findUserLicense(ctx context.Context, project, location, userID, licenseConfig string) (*geminiUserLicenseEntry, bool, error) {
	baseURL := r.userStoreBase(project, location) + "/userLicenses"
	pageToken := ""

	for {
		listURL := baseURL
		if pageToken != "" {
			listURL += "?pageToken=" + pageToken
		}

		body, err := r.doRequest(ctx, http.MethodGet, listURL, nil)
		if err != nil {
			if isGeminiNotFound(err) {
				return nil, false, nil
			}
			return nil, false, err
		}

		tflog.Debug(ctx, "userLicenses list response", map[string]any{"body": string(body)})

		var page listUserLicensesResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, false, fmt.Errorf("parsing userLicenses response: %w", err)
		}

		for i := range page.UserLicenses {
			tflog.Debug(ctx, "userLicense entry", map[string]any{
				"user":          page.UserLicenses[i].UserPrincipal,
				"licenseConfig": page.UserLicenses[i].LicenseConfig,
				"state":         page.UserLicenses[i].LicenseAssignmentState,
			})
			if page.UserLicenses[i].UserPrincipal == userID && page.UserLicenses[i].LicenseConfig == licenseConfig {
				return &page.UserLicenses[i], true, nil
			}
		}

		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}

	return nil, false, nil
}

func geminiLicenseID(project, location, userID string) string {
	return fmt.Sprintf("%s/%s/%s", project, location, userID)
}
