# Testing

Tests in Evergreen run on `cloud-dev` by default. To make them run in `cloud-qa` instead, in case `cloud-dev` is having issues for instance, the project variable `atlas_test_env` must be set to `QA`. Any other value will fallback to `cloud-dev`.

Those settings can be found in the evergreen site, selecting project `atlas-cli-plugin-kubernetes`, under the menu option `Variables`.
