package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gitlab "github.com/xanzy/go-gitlab"
)

const encoding = "base64"

func resourceGitlabRepositoryFile() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: `This resource allows you to create and manage GitLab repository files

**Limitations**:

The [GitLab Repository Files API](https://docs.gitlab.com/ee/api/repository_files.html)
can only create, update or delete a single file at the time.
The API will also
[fail with a 400](https://docs.gitlab.com/ee/api/repository_files.html#update-existing-file-in-repository)
response status code if the underlying repository is changed while the API tries to make changes.
Therefore, it's recommended to make sure that you execute it with
[-parallelism=1](https://www.terraform.io/docs/cli/commands/apply.html#parallelism-n)
and that no other entity than the terraform at hand makes changes to the
underlying repository while it's executing.
		`,

		CreateContext: resourceGitlabRepositoryFileCreate,
		ReadContext:   resourceGitlabRepositoryFileRead,
		UpdateContext: resourceGitlabRepositoryFileUpdate,
		DeleteContext: resourceGitlabRepositoryFileDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				s := strings.Split(d.Id(), ":")

				if len(s) != 3 {
					d.SetId("")
					return nil, fmt.Errorf("invalid Repository File import format; expected '{project_id}:{branch}:{file_path}'")
				}
				project, branch, filePath := s[0], s[1], s[2]

				d.SetId(filePath)
				d.Set("project", project)
				d.Set("branch", branch)

				return []*schema.ResourceData{d}, nil
			},
		},

		// the schema matches https://docs.gitlab.com/ee/api/repository_files.html#create-new-file-in-repository
		// However, we don't support the `encoding` parameter as it seems to be broken.
		// Only a value of `base64` is supported, all others, including the documented default `text`, lead to
		// a `400 {error: encoding does not have a valid value}` error.
		Schema: map[string]*schema.Schema{
			"project": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the project.",
			},
			"file_path": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The full path of the file. It must be relative to the root of the project without a leading slash `/`.",
			},
			"branch": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the branch to which to commit to.",
			},
			"start_branch": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the branch to start the new commit from.",
			},
			"author_email": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The email address of the commit author.",
			},
			"author_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the commit author.",
			},
			"content": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateBase64Content,
				Description:  "The content of the file. It must be base64 encoded.",
			},
			"commit_message": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The commit message.",
			},
			"overwrite_on_create": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If the file should be overwritten if it does already exist in the repository but not in the state.",
			},
		},
	}
}

func resourceGitlabRepositoryFileCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	filePath := d.Get("file_path").(string)

	var existingRepositoryFile *gitlab.File
	if d.Get("overwrite_on_create").(bool) {
		readOptions := &gitlab.GetFileOptions{
			Ref: gitlab.String(d.Get("branch").(string)),
		}

		existingRepositoryFile, _, _ = client.RepositoryFiles.GetFile(project, filePath, readOptions)
	}

	var filePathForId string
	if existingRepositoryFile == nil {
		options := &gitlab.CreateFileOptions{
			Branch:        gitlab.String(d.Get("branch").(string)),
			Encoding:      gitlab.String(encoding),
			AuthorEmail:   gitlab.String(d.Get("author_email").(string)),
			AuthorName:    gitlab.String(d.Get("author_name").(string)),
			Content:       gitlab.String(d.Get("content").(string)),
			CommitMessage: gitlab.String(d.Get("commit_message").(string)),
		}
		if startBranch, ok := d.GetOk("start_branch"); ok {
			options.StartBranch = gitlab.String(startBranch.(string))
		}

		repositoryFile, _, err := client.RepositoryFiles.CreateFile(project, filePath, options)
		if err != nil {
			return diag.FromErr(err)
		}
		filePathForId = repositoryFile.FilePath
	} else {
		options := &gitlab.UpdateFileOptions{
			Branch:        gitlab.String(d.Get("branch").(string)),
			Encoding:      gitlab.String(encoding),
			AuthorEmail:   gitlab.String(d.Get("author_email").(string)),
			AuthorName:    gitlab.String(d.Get("author_name").(string)),
			Content:       gitlab.String(d.Get("content").(string)),
			CommitMessage: gitlab.String(d.Get("commit_message").(string)),
			LastCommitID:  gitlab.String(existingRepositoryFile.LastCommitID),
		}
		if startBranch, ok := d.GetOk("start_branch"); ok {
			options.StartBranch = gitlab.String(startBranch.(string))
		}

		repositoryFile, _, err := client.RepositoryFiles.UpdateFile(project, filePath, options)
		if err != nil {
			return diag.FromErr(err)
		}
		filePathForId = repositoryFile.FilePath
	}

	d.SetId(filePathForId)
	return resourceGitlabRepositoryFileRead(ctx, d, meta)
}

func resourceGitlabRepositoryFileRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	filePath := d.Id()
	options := &gitlab.GetFileOptions{
		Ref: gitlab.String(d.Get("branch").(string)),
	}

	repositoryFile, _, err := client.RepositoryFiles.GetFile(project, filePath, options)
	if err != nil {
		if strings.Contains(err.Error(), "404 File Not Found") {
			log.Printf("[WARN] file %s not found, removing from state", filePath)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("project", project)
	d.Set("file_path", repositoryFile.FilePath)
	d.Set("branch", repositoryFile.Ref)
	d.Set("encoding", repositoryFile.Encoding)
	d.Set("content", repositoryFile.Content)

	return nil
}

func resourceGitlabRepositoryFileUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	filePath := d.Get("file_path").(string)

	readOptions := &gitlab.GetFileOptions{
		Ref: gitlab.String(d.Get("branch").(string)),
	}

	existingRepositoryFile, _, err := client.RepositoryFiles.GetFile(project, filePath, readOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.UpdateFileOptions{
		Branch:        gitlab.String(d.Get("branch").(string)),
		Encoding:      gitlab.String(encoding),
		AuthorEmail:   gitlab.String(d.Get("author_email").(string)),
		AuthorName:    gitlab.String(d.Get("author_name").(string)),
		Content:       gitlab.String(d.Get("content").(string)),
		CommitMessage: gitlab.String(d.Get("commit_message").(string)),
		LastCommitID:  gitlab.String(existingRepositoryFile.LastCommitID),
	}
	if startBranch, ok := d.GetOk("start_branch"); ok {
		options.StartBranch = gitlab.String(startBranch.(string))
	}

	_, _, err = client.RepositoryFiles.UpdateFile(project, filePath, options)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabRepositoryFileRead(ctx, d, meta)
}

func resourceGitlabRepositoryFileDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)
	project := d.Get("project").(string)
	filePath := d.Get("file_path").(string)

	readOptions := &gitlab.GetFileOptions{
		Ref: gitlab.String(d.Get("branch").(string)),
	}

	existingRepositoryFile, _, err := client.RepositoryFiles.GetFile(project, filePath, readOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	options := &gitlab.DeleteFileOptions{
		Branch:        gitlab.String(d.Get("branch").(string)),
		AuthorEmail:   gitlab.String(d.Get("author_email").(string)),
		AuthorName:    gitlab.String(d.Get("author_name").(string)),
		CommitMessage: gitlab.String(fmt.Sprintf("[DELETE]: %s", d.Get("commit_message").(string))),
		LastCommitID:  gitlab.String(existingRepositoryFile.LastCommitID),
	}

	resp, err := client.RepositoryFiles.DeleteFile(project, filePath, options)
	if err != nil {
		return diag.Errorf("%s failed to delete repository file: (%s) %v", d.Id(), resp.Status, err)
	}

	return nil
}

func validateBase64Content(v interface{}, k string) (we []string, errors []error) {
	content := v.(string)
	if _, err := base64.StdEncoding.DecodeString(content); err != nil {
		errors = append(errors, fmt.Errorf("given repository file content '%s' is not base64 encoded, but must be", content))
	}
	return
}
