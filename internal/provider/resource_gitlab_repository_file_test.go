package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "github.com/xanzy/go-gitlab"
)

func TestAccGitlabRepositoryFile_create(t *testing.T) {
	var file gitlab.File
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGitlabRepositoryFileConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3c=",
					}),
				),
			},
		},
	})
}

func TestAccGitlabRepositoryFile_validationOfBase64Content(t *testing.T) {
	cases := []struct {
		givenContent           string
		expectedIsValidContent bool
	}{
		{
			givenContent:           "not valid base64",
			expectedIsValidContent: false,
		},
		{
			givenContent:           "bWVvdyBtZW93IG1lb3c=",
			expectedIsValidContent: true,
		},
	}

	for _, c := range cases {
		_, errs := validateBase64Content(c.givenContent, "dummy")
		if len(errs) > 0 == c.expectedIsValidContent {
			t.Fatalf("content '%s' was either expected to be valid base64 but isn't or to be invalid base64 but actually is", c.givenContent)
		}
	}
}

func TestAccGitlabRepositoryFile_createOnNewBranch(t *testing.T) {
	var file gitlab.File
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGitlabRepositoryFileStartBranchConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3c=",
					}),
				),
				// see https://gitlab.com/gitlab-org/gitlab/-/issues/342200
				SkipFunc: func() (bool, error) {
					return true, nil
				},
			},
		},
	})
}

func TestAccGitlabRepositoryFile_update(t *testing.T) {
	var file gitlab.File
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGitlabRepositoryFileConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3c=",
					}),
				),
			},
			{
				Config: testAccGitlabRepositoryFileUpdateConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg==",
					}),
				),
			},
		},
	})
}

func TestAccGitlabRepositoryFile_overwriteExisting(t *testing.T) {
	var file gitlab.File
	rInt := acctest.RandInt()
	filePath := "meow.txt"

	testAccProvider, _ := providerFactories["gitlab-repository-files"]()

	// setup function to test when project is managed outside of terraform
	projectId, err := func() (int, error) {
		client := testAccProvider.Meta().(*gitlab.Client)

		createProjectOptions := &gitlab.CreateProjectOptions{
			Name:                 gitlab.String(fmt.Sprintf("foo-%d", rInt)),
			InitializeWithReadme: gitlab.Bool(true),
		}

		project, _, err := client.Projects.CreateProject(createProjectOptions)
		if err != nil {
			return -1, fmt.Errorf("setup failed: failed to create project: %s", err)
		}

		createFileOptions := &gitlab.CreateFileOptions{
			Branch:        gitlab.String("main"),
			Encoding:      gitlab.String("base64"),
			AuthorEmail:   gitlab.String("test"),
			AuthorName:    gitlab.String("test"),
			Content:       gitlab.String("dGVzdA=="),
			CommitMessage: gitlab.String("test"),
		}

		_, _, err = client.RepositoryFiles.CreateFile(project.ID, filePath, createFileOptions)
		if err != nil {
			return -1, fmt.Errorf("setup failed: failed to create file: %s", err)
		}

		return project.ID, nil
	}()

	if err != nil {
		fmt.Printf("setup failed: %s\n", err)
		return
	}

	defer func(projectId int) {
		client := testAccProvider.Meta().(*gitlab.Client)

		_, err := client.Projects.DeleteProject(projectId, nil)
		if err != nil {
			fmt.Printf("teardown failed: failed to delete project: %s\n", err)
		}
	}(projectId)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGitlabRepositoryFileOverwriteExistingConfig(projectId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabRepositoryFileExists("gitlab_repository_file.this_overwrite", &file),
					testAccCheckGitlabRepositoryFileAttributes(&file, &testAccGitlabRepositoryFileAttributes{
						FilePath: "meow.txt",
						Content:  "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg==",
					}),
				),
			},
		},
	})
}

func TestAccGitlabRepositoryFile_import(t *testing.T) {
	rInt := acctest.RandInt()
	resourceName := "gitlab_repository_file.this"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckGitlabRepositoryFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGitlabRepositoryFileConfig(rInt),
			},
			{
				ResourceName:            resourceName,
				ImportStateIdFunc:       getRepositoryFileImportID(resourceName),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"author_email", "author_name", "commit_message"},
			},
		},
	})
}

func getRepositoryFileImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("Not Found: %s", n)
		}

		repositoryFileID := rs.Primary.ID
		if repositoryFileID == "" {
			return "", fmt.Errorf("No repository file ID is set")
		}
		projectID := rs.Primary.Attributes["project"]
		if projectID == "" {
			return "", fmt.Errorf("No project ID is set")
		}
		branch := rs.Primary.Attributes["branch"]
		if branch == "" {
			return "", fmt.Errorf("No branch is set")
		}

		return fmt.Sprintf("%s:%s:%s", projectID, branch, repositoryFileID), nil
	}
}

func testAccCheckGitlabRepositoryFileExists(n string, file *gitlab.File) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		fileID := rs.Primary.ID
		branch := rs.Primary.Attributes["branch"]
		if branch == "" {
			return fmt.Errorf("No branch set")
		}
		options := &gitlab.GetFileOptions{
			Ref: gitlab.String(branch),
		}
		repoName := rs.Primary.Attributes["project"]
		if repoName == "" {
			return fmt.Errorf("No project ID set")
		}

		testAccProvider, _ := providerFactories["gitlab-repository-files"]()

		conn := testAccProvider.Meta().(*gitlab.Client)

		gotFile, _, err := conn.RepositoryFiles.GetFile(repoName, fileID, options)
		if err != nil {
			return fmt.Errorf("Cannot get file: %v", err)
		}

		if gotFile.FilePath == fileID {
			*file = *gotFile
			return nil
		}
		return fmt.Errorf("File does not exist")
	}
}

type testAccGitlabRepositoryFileAttributes struct {
	FilePath string
	Content  string
}

func testAccCheckGitlabRepositoryFileAttributes(got *gitlab.File, want *testAccGitlabRepositoryFileAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if got.FileName != want.FilePath {
			return fmt.Errorf("got name %q; want %q", got.FileName, want.FilePath)
		}

		if got.Content != want.Content {
			return fmt.Errorf("got content %q; want %q", got.Content, want.Content)
		}
		return nil
	}
}

func testAccCheckGitlabRepositoryFileDestroy(s *terraform.State) error {
	testAccProvider, _ := providerFactories["gitlab-repository-files"]()
	conn := testAccProvider.Meta().(*gitlab.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project" {
			continue
		}

		gotRepo, resp, err := conn.Projects.GetProject(rs.Primary.ID, nil)
		if err == nil {
			if gotRepo != nil && fmt.Sprintf("%d", gotRepo.ID) == rs.Primary.ID {
				if gotRepo.MarkedForDeletionAt == nil {
					return fmt.Errorf("Repository still exists")
				}
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

func testAccGitlabRepositoryFileConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name = "foo-%d"
  description = "Terraform acceptance tests"

  default_branch = "main"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
  initialize_with_readme = true
}

resource "gitlab_repository_file" "this" {
  project = "${gitlab_project.foo.id}"
  file_path = "meow.txt"
  branch = "main"
  content = "bWVvdyBtZW93IG1lb3c="
  author_email = "meow@catnip.com"
  author_name = "Meow Meowington"
  commit_message = "feature: add launch codes"
}
	`, rInt)
}

func testAccGitlabRepositoryFileStartBranchConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name = "foo-%d"
  description = "Terraform acceptance tests"

  default_branch = "main"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
  initialize_with_readme = true
}

resource "gitlab_repository_file" "this" {
  project = "${gitlab_project.foo.id}"
  file_path = "meow.txt"
  branch = "main"
  start_branch = "meow-branch"
  content = "bWVvdyBtZW93IG1lb3c="
  author_email = "meow@catnip.com"
  author_name = "Meow Meowington"
  commit_message = "feature: add launch codes"
}
	`, rInt)
}

func testAccGitlabRepositoryFileUpdateConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name = "foo-%d"
  description = "Terraform acceptance tests"

  default_branch = "main"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
  initialize_with_readme = true
}

resource "gitlab_repository_file" "this" {
  project = "${gitlab_project.foo.id}"
  file_path = "meow.txt"
  branch = "main"
  content = "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg=="
  author_email = "meow@catnip.com"
  author_name = "Meow Meowington"
  commit_message = "feature: change launch codes"
}
	`, rInt)
}

func testAccGitlabRepositoryFileOverwriteExistingConfig(projectId int) string {
	return fmt.Sprintf(`
resource "gitlab_repository_file" "this_overwrite" {
  project = "%d"
  file_path = "meow.txt"
  branch = "main"
  content = "bWVvdyBtZW93IG1lb3cgbWVvdyBtZW93Cg=="
  author_email = "meow@catnip.com"
  author_name = "Meow Meowington"
  commit_message = "feature: overwrite launch codes"
  overwrite_on_create = true
}
	`, projectId)
}
