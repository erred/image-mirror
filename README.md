# image-mirror

A repo for image-mirror

[![License](https://img.shields.io/github/license/seankhliao/image-mirror.svg?style=flat-square)](LICENSE)

## Unimplemented

- use imagepullsecrets
  - necessary to work with private upstreams
- use patch?
  - tried with `client.Patch` but it didn't stick, need more investigation
- gracefully handle retries
  - this is probably easier with patch
- write tests?
  - skaffold dev + "works on my machine"
- validate deployment
  - not clear what "you should be able to get the controller to re-deploy" means
    - maybe same as retries?
  - currently panics (intentionally) with unimplemented
- lock down security
  - use non root
    - need to create home

## Requirements

- docker with buildkit enabled

tested with:

- go1.16beta1 and tip
- kind v0.10.0
- kubectl v0.20.2
- kubernetes 1.20.2
- kustomize v3.9.2
- skaffold v1.18.0
