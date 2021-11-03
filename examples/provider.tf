terraform {
  required_providers {
    gitlab = {
      source  = "gitlabhq/gitlab"
      version = "3.7.0"
    }
    gitlab-repository-files = {
      source  = "timofurrer/gitlab-repository-files"
      version = "0.4.3"
    }
  }
}

provider "gitlab-repository-files" {
  # Configuration options
  base_url = var.base_url
  token    = var.gitlab_token
}

provider "gitlab" {
  # Configuration options
  base_url = var.base_url
  token    = var.gitlab_token
}
