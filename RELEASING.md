# Releasing

We generate packages via the `package_goreleaser` evergreen task. This task runs off of main and is only triggered when a new tag is pushed.

### Release Steps
To manually generate a new stable release you can run:


```bash
git tag -a -s "v1.0.0" -m "v1.0.0"
git push origin "v1.0.0"
```

**Note:** Please use the `vX.Y.Z` format for the version to release.

This will do the following things:
1. The [evergreen](build/ci/release.yml) release task will run after a tag event from main.
2. This task signs all packages and includes both them and the public key in the release.
3. If everything goes smoothly, the release will be published in the [releases page](https://github.com/mongodb/atlas-cli-plugin-kubernetes/releases).
4. The [evergreen](build/ci/release.yml) copybara task will automatically open a PR on docs repositories with any document changes for the docs team to review and merge. 
