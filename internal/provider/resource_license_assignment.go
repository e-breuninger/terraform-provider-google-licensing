package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/licensing/v1"
)

var (
	_ resource.Resource                = &licenseAssignmentResource{}
	_ resource.ResourceWithImportState = &licenseAssignmentResource{}
	_ resource.ResourceWithConfigure   = &licenseAssignmentResource{}
)

func NewLicenseAssignmentResource() resource.Resource {
	return &licenseAssignmentResource{}
}

type licenseAssignmentResource struct {
	client *licensing.Service
}

type licenseAssignmentModel struct {
	ID          types.String `tfsdk:"id"`
	ProductID   types.String `tfsdk:"product_id"`
	SKUID       types.String `tfsdk:"sku_id"`
	UserID      types.String `tfsdk:"user_id"`
	ProductName types.String `tfsdk:"product_name"`
	SKUName     types.String `tfsdk:"sku_name"`
}

func (r *licenseAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_assignment"
}

func (r *licenseAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Assigns a Google Workspace (Enterprise) license SKU to a user.

The resource maps to the [Enterprise License Manager API](https://developers.google.com/admin-sdk/licensing/reference/rest/v1/licenseAssignments).

## Import

Existing assignments can be imported using the composite ID:

` + "```" + `shell
terraform import googleenterpriselicense_assignment.example <product_id>/<sku_id>/<user_id>
` + "```",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the form `<product_id>/<sku_id>/<user_id>`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_id": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The product's unique identifier. " +
					"Common values: `Google-Apps`, `Google-Vault`, `Google-Drive-storage`, `Google-Coordinate`." +
					" See the [product and SKU IDs reference](https://developers.google.com/admin-sdk/licensing/v1/how-tos/products).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sku_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The SKU's unique identifier within the product.",
			},
			"user_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The user's primary email address or unique user ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"product_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The human-readable display name of the product, as returned by the API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sku_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The human-readable display name of the SKU, as returned by the API.",
			},
		},
	}
}

func (r *licenseAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = pc.Licensing
}

func (r *licenseAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan licenseAssignmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	productID := plan.ProductID.ValueString()
	skuID := plan.SKUID.ValueString()
	userID := plan.UserID.ValueString()

	tflog.Debug(ctx, "Creating license assignment", map[string]any{
		"product_id": productID,
		"sku_id":     skuID,
		"user_id":    userID,
	})

	body := &licensing.LicenseAssignmentInsert{UserId: userID}
	assignment, err := r.client.LicenseAssignments.Insert(productID, skuID, body).Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating license assignment",
			fmt.Sprintf("Could not assign license (product=%s, sku=%s, user=%s): %s", productID, skuID, userID, err),
		)
		return
	}

	plan.ID = types.StringValue(assignmentID(productID, skuID, userID))
	plan.ProductName = types.StringValue(assignment.ProductName)
	plan.SKUName = types.StringValue(assignment.SkuName)

	tflog.Trace(ctx, "Created license assignment", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *licenseAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state licenseAssignmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	productID := state.ProductID.ValueString()
	skuID := state.SKUID.ValueString()
	userID := state.UserID.ValueString()

	tflog.Debug(ctx, "Reading license assignment", map[string]any{
		"product_id": productID,
		"sku_id":     skuID,
		"user_id":    userID,
	})

	assignment, err := r.client.LicenseAssignments.Get(productID, skuID, userID).Context(ctx).Do()
	if err != nil {
		if isNotFound(err) {
			tflog.Warn(ctx, "License assignment not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading license assignment",
			fmt.Sprintf("Could not read license assignment %s: %s", state.ID.ValueString(), err),
		)
		return
	}

	state.ProductID = types.StringValue(assignment.ProductId)
	state.SKUID = types.StringValue(assignment.SkuId)
	state.UserID = types.StringValue(assignment.UserId)
	state.ProductName = types.StringValue(assignment.ProductName)
	state.SKUName = types.StringValue(assignment.SkuName)
	state.ID = types.StringValue(assignmentID(assignment.ProductId, assignment.SkuId, assignment.UserId))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *licenseAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state licenseAssignmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	productID := state.ProductID.ValueString()
	oldSKUID := state.SKUID.ValueString()
	newSKUID := plan.SKUID.ValueString()
	userID := state.UserID.ValueString()

	tflog.Debug(ctx, "Updating license assignment SKU", map[string]any{
		"product_id": productID,
		"old_sku_id": oldSKUID,
		"new_sku_id": newSKUID,
		"user_id":    userID,
	})

	body := &licensing.LicenseAssignment{SkuId: newSKUID}
	assignment, err := r.client.LicenseAssignments.Patch(productID, oldSKUID, userID, body).Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating license assignment",
			fmt.Sprintf("Could not patch SKU from %s to %s for user %s: %s", oldSKUID, newSKUID, userID, err),
		)
		return
	}

	plan.ID = types.StringValue(assignmentID(assignment.ProductId, assignment.SkuId, assignment.UserId))
	plan.ProductID = types.StringValue(assignment.ProductId)
	plan.SKUID = types.StringValue(assignment.SkuId)
	plan.UserID = types.StringValue(assignment.UserId)
	plan.ProductName = types.StringValue(assignment.ProductName)
	plan.SKUName = types.StringValue(assignment.SkuName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *licenseAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state licenseAssignmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	productID := state.ProductID.ValueString()
	skuID := state.SKUID.ValueString()
	userID := state.UserID.ValueString()

	tflog.Debug(ctx, "Deleting license assignment", map[string]any{
		"product_id": productID,
		"sku_id":     skuID,
		"user_id":    userID,
	})

	_, err := r.client.LicenseAssignments.Delete(productID, skuID, userID).Context(ctx).Do()
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting license assignment",
			fmt.Sprintf("Could not delete license assignment %s: %s", state.ID.ValueString(), err),
		)
	}
}

func (r *licenseAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format <product_id>/<sku_id>/<user_id>, got: %q", req.ID),
		)
		return
	}

	state := licenseAssignmentModel{
		ID:        types.StringValue(req.ID),
		ProductID: types.StringValue(parts[0]),
		SKUID:     types.StringValue(parts[1]),
		UserID:    types.StringValue(parts[2]),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func assignmentID(productID, skuID, userID string) string {
	return fmt.Sprintf("%s/%s/%s", productID, skuID, userID)
}

func isNotFound(err error) bool {
	if apiErr, ok := err.(*googleapi.Error); ok {
		return apiErr.Code == 404
	}
	return false
}
