# Terraform module - S3 bucket
## Example
``` hcl
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.14.0"
    }
  }
}

provider "aws" {
  alias  = "frankfurt"
  region = "eu-central-1"
}

module "bucket" {
  providers = {
    aws = aws.frankfurt
  }
  # Use v1 to use the latest build from major version v1
  # Use a specific tag to lock down the version to a specific commit such as v1.0.7
  source = "git::https://github.com/rbjoergensen/terraform-s3.git?ref=v1"
  bucket_name = "callofthevoid-test-bucket"
}
```
