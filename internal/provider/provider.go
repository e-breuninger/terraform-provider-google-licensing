package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
	"google.golang.org/api/licensing/v1"
	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
)

var (
	_ provider.Provider              = &googleEnterpriseLicenseProvider{}
	_ provider.ProviderWithFunctions = &googleEnterpriseLicenseProvider{}
)

const (
	licensingScope     = "https://www.googleapis.com/auth/apps.licensing"
	cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"
)

type providerClient struct {
	Licensing  *licensing.Service
	HTTPClient *http.Client
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &googleEnterpriseLicenseProvider{version: version}
	}
}

type googleEnterpriseLicenseProvider struct {
	version string
}

type providerModel struct {
	Credentials     types.String `tfsdk:"credentials"`
	CredentialsFile types.String `tfsdk:"credentials_file"`
	AccessToken     types.String `tfsdk:"access_token"`
}

func (p *googleEnterpriseLicenseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "googleenterpriselicense"
	resp.Version = p.version
}

func (p *googleEnterpriseLicenseProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The **Google Enterprise License Manager** provider manages Google Workspace license
assignments via the [Enterprise License Manager API](https://developers.google.com/admin-sdk/licensing)
and Gemini Enterprise (NotebookLM) licenses via the Discovery Engine API.

## Authentication

The provider supports four credential strategies, evaluated in order:

1. **access_token** – a pre-fetched OAuth2 access token, e.g. minted for an impersonated
   service account via the ` + "`google_service_account_access_token`" + ` data source of the
   official ` + "`google`" + ` provider.
2. **credentials** – inline JSON of a service-account key (or user credentials).
3. **credentials_file** – path to a JSON credentials file.
4. **Application Default Credentials (ADC)** – used automatically when none of the above is set.`,

		Attributes: map[string]schema.Attribute{
			"credentials": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "JSON content of a Google service-account key file. Can also be set via the `GOOGLE_CREDENTIALS` environment variable.",
			},
			"credentials_file": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Path to a JSON credentials file. Can also be set via the `GOOGLE_APPLICATION_CREDENTIALS` environment variable.",
			},
			"access_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "A pre-fetched OAuth2 access token, typically minted for an impersonated service account (e.g. via the `google_service_account_access_token` data source). Takes precedence over `credentials` and `credentials_file`.",
			},
		},
	}
}

func (p *googleEnterpriseLicenseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentialsJSON := resolveStringValue(config.Credentials, "GOOGLE_CREDENTIALS")
	credentialsFile := resolveStringValue(config.CredentialsFile, "GOOGLE_APPLICATION_CREDENTIALS")
	accessToken := resolveStringValue(config.AccessToken, "GOOGLE_OAUTH_ACCESS_TOKEN")

	client, err := buildProviderClient(accessToken, credentialsJSON, credentialsFile)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build API clients", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *googleEnterpriseLicenseProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewLicenseAssignmentResource,
		NewGeminiUserLicenseResource,
	}
}

func (p *googleEnterpriseLicenseProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *googleEnterpriseLicenseProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}

func buildProviderClient(accessToken, credentialsJSON, credentialsFile string) (*providerClient, error) {
	bgCtx := context.Background()

	var licensingOpts, httpOpts []option.ClientOption

	switch {
	case accessToken != "":
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
		licensingOpts = []option.ClientOption{
			option.WithTokenSource(tokenSource),
			option.WithScopes(licensingScope),
		}
		httpOpts = []option.ClientOption{
			option.WithTokenSource(tokenSource),
			option.WithScopes(cloudPlatformScope),
		}

	case credentialsJSON != "":
		if !json.Valid([]byte(credentialsJSON)) {
			return nil, fmt.Errorf("credentials value is not valid JSON")
		}
		licensingOpts = []option.ClientOption{
			option.WithCredentialsJSON([]byte(credentialsJSON)),
			option.WithScopes(licensingScope),
		}
		httpOpts = []option.ClientOption{
			option.WithCredentialsJSON([]byte(credentialsJSON)),
			option.WithScopes(cloudPlatformScope),
		}

	case credentialsFile != "":
		licensingOpts = []option.ClientOption{
			option.WithCredentialsFile(credentialsFile),
			option.WithScopes(licensingScope),
		}
		httpOpts = []option.ClientOption{
			option.WithCredentialsFile(credentialsFile),
			option.WithScopes(cloudPlatformScope),
		}

	default:
		httpOpts = []option.ClientOption{option.WithScopes(cloudPlatformScope)}
	}

	licensingSvc, err := licensing.NewService(bgCtx, licensingOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating licensing service: %w", err)
	}

	httpClient, _, err := htransport.NewClient(bgCtx, httpOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP client for Gemini Enterprise API: %w", err)
	}

	return &providerClient{
		Licensing:  licensingSvc,
		HTTPClient: httpClient,
	}, nil
}

func resolveStringValue(v types.String, envVar string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	return os.Getenv(envVar)
}
