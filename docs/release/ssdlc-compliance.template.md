SSDLC Compliance Report: Atlas CLI Plugin Kubernetes ${VERSION}
=================================================================

- Release Creator: ${AUTHOR}
- Created On:      ${DATE}

Overview:

- **Product and Release Name**
    - Atlas CLI Plugin Kubernetes ${VERSION}, ${DATE}.

- **Process Document**
  - https://www.mongodb.com/blog/post/how-mongodb-protects-against-supply-chain-vulnerabilities

- **Tool used to track third party vulnerabilities**
  - [Kondukto](https://arcticglow.kondukto.io/)

- **Dependency Information**
  - As part of every release, both the SBOM and the Augmented SBOM are generated and published alongside the release artifacts.
  - These files provide a comprehensive view of third-party dependencies and their associated metadata, enabling compliance tracking.
  - The Augmented SBOM is further enriched with vulnerability scanning data sourced from Kondukto.
    - [Download SBOM](https://github.com/mongodb/atlas-cli-plugin-kubernetes/releases/download/v${VERSION}/sbom.json)
    - [Download Augmented SBOM](https://github.com/mongodb/atlas-cli-plugin-kubernetes/releases/download/v${VERSION}/augmented-sbom.json)

- **Security Testing Report**
  - Available as needed from Cloud Security.

- **Security Assessment Report**
  - Available as needed from Cloud Security.

Assumptions and attestations:

- Internal processes are used to ensure CVEs are identified and mitigated within SLAs.