resource "gitlab_project" "foo" {
  name        = "foo"
  description = "Example"

  default_branch = "main"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level       = "internal"
  initialize_with_readme = true
}

resource "gitlab-repository-files_gitlab_repository_file" "this" {
  project        = gitlab_project.foo.id
  file_path      = "meow.txt"
  branch         = "main"
  content        = base64encode("hello world")
  author_email   = "meow@catnip.com"
  author_name    = "Meow Meowington"
  commit_message = "feature: add launch codes"
}
