package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"token": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("GITLAB_TOKEN", nil),
					Description: "The OAuth2 token or project/personal access token used to connect to GitLab.",
				},
				"base_url": {
					Type:         schema.TypeString,
					Optional:     true,
					DefaultFunc:  schema.EnvDefaultFunc("GITLAB_BASE_URL", ""),
					Description:  "The GitLab Base API URL",
					ValidateFunc: validateApiURLVersion,
				},
				"cacert_file": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: "A file containing the ca certificate to use in case ssl certificate is not from a standard chain",
				},
				"insecure": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Disable SSL verification of API calls",
				},
				"client_cert": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: "File path to client certificate when GitLab instance is behind company proxy. File  must contain PEM encoded data.",
				},
				"client_key": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: "File path to client key when GitLab instance is behind company proxy. File must contain PEM encoded data.",
				},
			},

			ResourcesMap: map[string]*schema.Resource{
				"gitlabx_repository_file": resourceGitlabRepositoryFile(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		config := Config{
			Token:      d.Get("token").(string),
			BaseURL:    d.Get("base_url").(string),
			CACertFile: d.Get("cacert_file").(string),
			Insecure:   d.Get("insecure").(bool),
			ClientCert: d.Get("client_cert").(string),
			ClientKey:  d.Get("client_key").(string),
		}

		client, err := config.Client()
		if err != nil {
			return nil, diag.FromErr(err)
		}

		userAgent := p.UserAgent("terraform-provider-gitlab-repository-files", version)
		client.UserAgent = userAgent

		return client, diag.FromErr(err)
	}
}

func validateApiURLVersion(value interface{}, key string) (ws []string, es []error) {
	v := value.(string)
	if strings.HasSuffix(v, "/api/v3") || strings.HasSuffix(v, "/api/v3/") {
		es = append(es, fmt.Errorf("terraform-provider-gitlab-repository-files does not support v3 api; please upgrade to /api/v4 in %s", v))
	}
	return
}
