---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "gitlab-repository-files Provider"
subcategory: ""
description: |-
  
---

# gitlab-repository-files Provider





<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- **base_url** (String) The GitLab Base API URL
- **cacert_file** (String) A file containing the ca certificate to use in case ssl certificate is not from a standard chain
- **client_cert** (String) File path to client certificate when GitLab instance is behind company proxy. File  must contain PEM encoded data.
- **client_key** (String) File path to client key when GitLab instance is behind company proxy. File must contain PEM encoded data.
- **insecure** (Boolean) Disable SSL verification of API calls
- **token** (String) The OAuth2 token or project/personal access token used to connect to GitLab.
