# Delegated storage

Created on 2021-12-09 by Paweł Kosiec ([@pkosiec](https://github.com/pkosiec/))

## Overview

This document describes the way how we will approach dynamic, external data for TypeInstances.

<!-- toc -->

- [Motivation](#motivation)
  * [Goal](#goal)
  * [Non-goal](#non-goal)
- [Assumptions](#assumptions)
- [Prerequisites](#prerequisites)
- [Registering storage backends](#registering-storage-backends)
- [Workflow syntax - Create](#workflow-syntax---create)
- [Workflow syntax - Update](#workflow-syntax---update)
- [Storage backend service implementation](#storage-backend-service-implementation)
- [Configuring storage backends](#configuring-storage-backends)
- [Uninstalling storage backends](#uninstalling-storage-backends)
- [GraphQL API](#graphql-api)
  * [List storage backends](#list-storage-backends)
  * [Get TypeInstance details](#get-typeinstance-details)
  * [TypeInstance create](#typeinstance-create)
  * [TypeInstance update](#typeinstance-update)
- [Dynamic TypeInstance projections](#dynamic-typeinstance-projections)
  * [Introduction](#introduction)
  * [Go Template backend storage](#go-template-backend-storage)
  * [Helm runner templating](#helm-runner-templating)
- [Rejected ideas](#rejected-ideas)
  * [Registering storage backends](#registering-storage-backends-1)
  * [Workflow syntax](#workflow-syntax)
  * [Storage backend service implementation](#storage-backend-service-implementation-1)
  * [Configuring default storage backends](#configuring-default-storage-backends)
- [Potential future evolution](#potential-future-evolution)
  * [Key points](#key-points)
  * [Summary](#summary)
- [Consequences](#consequences)

<!-- tocstop -->

## Motivation

Capact stores the state in a form of TypeInstances with static data. That is problematic, as the data may quickly become outdated in case of any external change. For example, if you install Mattermost using Capact and delete Helm chart with Helm CLI, you will still be able to see Helm release TypeInstance of already not existing Helm release.

Also, we should be able to provide a way to store sensitive data, such as credentials, securely. Currently, we store them as plaintext inside our database.

The last reason, is that long-term we should replace Neo4j database for a more lightweight solution (like sqlite or PostgreSQL). Apart from being resource-hungry, Neo4j can be problematic also when it comes to licensing (GPL3). We believe that the pluggable back-ends concept could be a first step to abstract Neo4j and plug-in different storage backend. However, this should be treated as an additional, nice-to-have goal, or side effect of the Delegated Storage proposal.

### Goal

The main goal is to support the following use-cases:

- Store and retrieve secrets using external, secure solutions
    - Examples:
        - user credentials (e.g. for PostgreSQL, GitLab)
        - SSH key (e.g. for bastion)
        - Kubeconfig (for external clusters)

    - The following backends should be supported initially (sorted by priority):
        1. Vault
        1. Secrets encrypted with [SOPS](https://github.com/mozilla/sops) stored on Git repository
        pluggable

        Other backends:
        - AWS Secrets Manager
        - Google Key Management
    
- Store and retrieve dynamic data
    - Examples:
        - Kubernetes cluster (e.g. Flux's Helm releases or Kubernetes Secrets)
        - Apps configuration (e.g. Mattermost config)
        - External dependencies (e.g. S3 buckets)

    - The following backends should be supported initially (sorted by priority):
        1. Flux HelmRelease Custom Resources
        1. Git repositories (e.g. GitLab projects)
        1. S3 buckets (e.g. for external Terraform state)

- Ability to manage such TypeInstances manually (via CLI and maybe UI) and as a part of Action
    - Support such dynamic TypeInstance creation, update, and deletion
    - Define GraphQL API and Implementation workflow syntax

- Support for automatic TypeInstance creation and deletion

- Support extensibility for upcoming backends

Also, the additional, nice-to-have goals are:

- Remove Neo4j dependency from Local Hub while preserving TypeInstance metadata, such as relations

### Non-goal

- Support external back-ends for Capact manifest storage (Public Hub)
- Remove Neo4j dependency from Public Hub

## Assumptions

1. Content Developer should be able to:
    1. Write manifests without specifying a storage backend (use default one configured by System Administrator). In this case, a static TypeInstance value is stored in the default storage backend.
    1. Specify a specific storage backend as a part of a given Implementation. This case supports both static and dynamic TypeInstance values.
1. System Administrator can configure default backend storage for static values.
1. There are two different cases when it comes CRUD operations on TypeInstances:
    1. CRUD operations on TypeInstance actually manages external resource (e.g. Vault). That is, CRUD operations on TypeInstances in Local Hub actually creates, updates and deletes a given resource.
    1. CRUD operations on TypeInstance represents external resources managed in different way (e.g. by running Helm install). That is, CRUD operations on TypeInstances in Local Hub actually registers, unregisters and updates references for external state without changing them.
1. Storage backend stores only TypeInstance value and locking state (`lockedBy` property). All other TypeInstance metadata, including relations, are stored in built-in Hub storage.

## Prerequisites

1. Implement these two Type features:
    - [Additional references to parent nodes](https://capact.io/docs/feature/type-features/#additional-references-to-parent-nodes)
    - [Find Types based on prefix of parent nodes](https://capact.io/docs/feature/type-features#find-types-based-on-prefix-of-parent-nodes)
1. [Add TypeInstance `alias` metadata field](https://github.com/capactio/capact/issues/579)
1. [Validate TypeInstance input](https://github.com/capactio/capact/issues/544)

## Registering storage backends

1. For every storage backend, we create a dedicated Type:

    ```yaml
    ocfVersion: 0.0.1
    revision: 0.0.1
    kind: Type
    metadata:
      path: cap.type.helm.storage
    spec:
      additionalRefs:
        - "cap.core.type.hub.storage" 
      jsonSchema:
        value: |- # JSON schema with:
          {
          "$schema": "http://json-schema.org/draft-07/schema",
          "type": "object",
          "title": "The root schema",
          "required": [
            "url",
            "contextSchema"
          ],
          "properties": {
            "url": { # url of hosted app, which implements storage backend ProtocolBuffers interface.
              "$id": "#/properties/url",
              "type": "string",
              "format": "uri"
            },
            "contextSchema": { # JSON schema which describes additional properties passed in Capact workflow
              "const": { # see http://json-schema.org/draft/2019-09/json-schema-validation.html#rfc.section.6.1.3
                "$schema": "http://json-schema.org/draft-07/schema",
                "type": "object",
                "required": [
                  "name",
                  "namespace"
                ],
                "properties": {
                  "name": {
                    "$id": "#/properties/context/properties/name",
                    "type": "string"
                  },
                  "namespace": {
                    "$id": "#/properties/context/properties/namespace",
                    "type": "string"
                  }
                },
                "additionalProperties": false
              }
            },
            "acceptValue": { # specifies if a given storage backend (app) accepts TypeInstance value while creating/updating TypeInstance, or just context
              "$id": "#/properties/acceptValue",
              "type": "boolean",
              "const": false # in this case - no
            },
          },
          "additionalProperties": false
        }
    ```

    It should follow the convention of having `url`, `acceptValue`, and `contextSchema` fields. As `context` are optional, the `contextSchema` field is nullable.

    We can validate such convention using custom logic for Type validation. In case of the `cap.core.hub.storage` additional reference, we could prevent uploading such Type if the JSON schema don't meet our conditions.

    > **NOTE:** See also the [Rejected ideas](#rejected-ideas) section to learn why a generic validation idea was rejected.

1. To install new storage backend, System Administrator has two options:

    - use Capact Actions (e.g. `cap.interface.capactio.capact.hub.storage.helm-release.install`).
    - Register a storage backend by creating such TypeInstance.

    Regardless the option, at the end there is one TypeInstance produced:
      
      ```yaml
      id: 3ef2e4ac-9070-4093-a3ce-142139fd4a16
      typeRef:
        path: cap.type.helm.storage
        revision: 0.1.0
      latestResourceVersion:
        metadata:
          alias: helm-storage # new field, more user-friendly description of such TypeInstance
          attributes:
          - path: "cap.core.attribute.hub.storage.backend" # related to GraphQL implementation; can be skipped after full support for additional references to parent nodes
            revision: 0.1.0
        value:
          url: "helm-release.default:50051"
          acceptValue: false
          contextSchema: |-
            {
              "$id": "#/properties/contextSchema",
              "type": "object",
              "required": [
                "name",
                "namespace"
              ],
              "properties": {
                "name": {
                  "$id": "#/properties/contextSchema/properties/name",
                  "type": "string"
                },
                "namespace": {
                  "$id": "#/properties/contextSchema/properties/namespace",
                  "type": "string"
                }
              },
              "additionalProperties": false
            }
      backend:
        id: "a36ed738-dfe7-45ec-acd1-8e44e8db893b" # new immutable property - contains TypeInstance ID
          # if not provided during TypeInstance creation, fallback to default one (get TypeInstance with proper Attribute and write its ID in this property)
      ```

1. In fresh Capact installation, there is one TypeInstance already preregistered:

    ```yaml
    id: a36ed738-dfe7-45ec-acd1-8e44e8db893b
    typeRef:
        path: cap.core.type.hub.storage.postgresql
        revision: 0.1.0
    latestResourceVersion:
      metadata:
        alias: capact-postgresql
        attributes:
        - path: cap.core.attribute.hub.storage.backend
          revision: 0.1.0
      value:
        # no url as it is known for Hub (Hub is already connected with PostgreSQL)
        acceptValue: true
        contextSchema: null
    backend: 
      abstract: true # Special keyword which specifies the built-in storage option which stores already all other metadata.
    ```

    - The one preregistered storage backend is Capact PostgreSQL. It uses special `backend` property: `abstract: true`.

      It is a reserved system property and no user can create such TypeInstances. We can enforce that with a custom check during TypeInstance creation on Hub GraphQL API to prevent creation of such TypeInstances.

      The property indicates that Hub queries the PostgreSQL database directly.

    - It is the default backend for static TypeInstance values. To learn more, read the [Configuring storage backends](#configuring-storage-backends) paragraph.    

## Workflow syntax - Create

1. In workflow, Content Developer can specify requirements for a given backend:

    ```yaml
    requires:
      cap.core.type.hub.storage: # Optional - Content Dev specifies such requirement to force use a given backend
        allOf:
          - typeRef:
              path: cap.type.helm.storage
              revision: 0.1.0
              alias: helm-storage
    ```

    - This workflow cannot be run unless there is a `helm-release` storage backend installed (where `helm-release` is only workflow alias).
    - If there are no specific storage backend requirements set, the default backend will be used. To learn more, read the [Configuring default storage backends](#configuring-default-storage-backends) paragraph.    

1. Content Developer outputs one of the following Argo workflow artifacts:

    > **NOTE:** Before this proposal, the whole Argo workflow artifact was treated as a value. Now we would need to change that.

    1. To store a given value on default backend or backend without any required additional parameters, which also accepts TypeInstance value:

        ```yaml
        # option 1: save a specific value on a storage backend
        # a given backend
        value: foo
        ```

    1. To point to some external data for a given storage backend:

        ```yaml
        # option 2: register something which already exist as external TypeInstance - based on `backend.context`
        backend:
          context:
            name: release-name
            namespace: release-namespace
        ```

        However, the `context` are backend-specific properties, which means Content Developer need to explicitly specify the backend as described later.

    1. To save a specific value with additional parameters:
    
        For example, for an implementation of Kubernetes secrets storage backend, which actually creates and updates these secrets during TypeInstance creation:

        ```yaml
        # option 3: save a specific value on an external backend with some additional parameters
        value: foo
        backend:
          context:
            key: bar
            value: baz
        ```

        The storage backend has to have `contextSchema` specified, as well as the `acceptValue` property set to `true`.

    In that way, someday we will be able to extend such approach with additional properties:
    
    ```yaml    
    instrumentation: # someday - if we want to unify the approach
      health:
        endpoint: foo.bar/healthz
      # (...)
    ```

    Such `instrumentation` data would be also stored in the same storage backend as the `value`. If Content Developer wants to store it somewhere else, then an additional Argo artifact to produce is needed.

1. Then, Content Developer specifies the Argo workflow artifact as output TypeInstance with familiar syntax:

    ```yaml
    # default - static
    capact-outputTypeInstances:
      - name: mattermost-config
        from: additional
        # no backend definition -> use default storage backend

    # option 2 - specific backend (defined in `Implementation.spec.requires` property)
    capact-outputTypeInstances:
      - name: helm-release
        from: helm-release
        backend: helm-storage # new property -> alias defined in the `requires` section
    ```

1. The automatically injected TypeInstance upload step, receives the following payload:

    ```yaml
    typeInstances:
    - alias: helm-release
      attributes: []
      createdBy: default/act-l49vh-30c7a078-6a77-475c-94dd-7466f56447ce
      typeRef:
        path: cap.type.helm.chart.release
        revision: 0.1.0
      value: null
      backend:
        id: 3ef2e4ac-9070-4093-a3ce-142139fd4a16 # helm-release backend - resolved UUID based on the injected TypeInstance

    usesRelations: # automatically create relation between TypeInstance using a given backend
    - from: helm-release
      to: 3ef2e4ac-9070-4093-a3ce-142139fd4a16
    ```

    The `helm-release` Argo artifact contains:

    ```yaml
    backend:
      # Fields which are a part of TypeInstanceResourceVersion (can be changed later via TypeInstance Update):
      context:
        name: release-name
        namespace: release-namespace
    ```

1. Hub receives the following GraphQL mutation based on the payload fields and artifact value from point above:

    ```graphql
    mutation CreateTypeInstances {
      createTypeInstances(
        in: {
         typeInstances: [
          {
            alias: "helm-release"
            typeRef: { path: "cap.type.helm.chart.release", revision: "0.1.0" }
            attributes: []
            value: {
              backend: {
                id: "3ef2e4ac-9070-4093-a3ce-142139fd4a16",
                context: {
                  name: "release-name",
                  namespace: "release-namespace",
                }
              }
            }
          }
        ],
        usesRelations: [
          {
            from: "helm-release",
            to: "3ef2e4ac-9070-4093-a3ce-142139fd4a16"
          }
        ]
      }
      ) {
        id
        alias
      }
    }
    ```

1. Based on the `backend` data:

    1. Hub resolves details about the service (TypeInstance details)
    1. If the TypeInstance value is provided:
        
        1. Hub checks whether the storage backend accepts TypeInstance value (`acceptValue` property). If not, and the value has been provided, it returns error.
        1. Hub validates the TypeInstance value against Type JSON Schema.

    1. Hub validates the `context` against JSON schema saved in the storage backend TypeInstance.
    1. Hub calls the registered storage backend service `onCreate` hook:

        ```proto
        message TypeInstanceData {
          string id = 1;
          bytes value = 2;
        }

        message OnCreateRequest {
          TypeInstanceData typeinstance = 1;
          bytes context = 2;
        }

        message OnCreateResponse {
          optional bytes context = 1;
        }

        service SearchService {
          rpc OnCreate(OnCreateRequest) returns (OnCreateResponse);
        }
        ```

        This hook can mutate `context`.

    1. Saves TypeInstance metadata in the core Hub storage backend, which contains all metadata of the TypeInstances and  theirs relations.

      ``` yaml
      id: 5d925774-55d3-4c74-8ecb-c984ea7ef636
      typeRef:
        path: cap.core.type.hub.storage.postgresql
        revision: 0.1.0
      latestResourceVersion:
        resourceVersion: 1
        backend:
          context: # additional parameters that might be modified via the service handling `onCreate` hook
            name: release-name
            namespace: release-namespace 
      backend: 
        id: 3ef2e4ac-9070-4093-a3ce-142139fd4a16 # helm-release backend - resolved UUID based on the injected TypeInstance
      ```

## Workflow syntax - Update

Similarly as with create, Content Developer specifies in the workflow:

```yaml
capact-updateTypeInstances:
- name: testUpdate
  from: update
```

where the `update` Argo artifact can contain `value` and / or `context`.

For additions in GraphQL API, see the [GraphQL API](#graphql-api) section.

## Storage backend service implementation

Capact Local Hub calls proper storage backend service while accessing the TypeInstance value or lock state.

1.  The registered storage backend service needs to implement the following gRPC + Protocol Buffers API:

    <details> <summary>Protocol Buffers definition</summary>

    ```protobuf
    syntax = "proto3";
    option go_package = "./storage_backend";
    package storage_backend;
    
    message OnCreateRequest {
      string typeinstance_id = 1;
      bytes value = 2;
      bytes context = 3;
    }
    
    message OnCreateResponse {
      optional bytes context = 1;
    }
    
    message TypeInstanceResourceVersion {
      uint32 resource_version = 1;
      bytes value = 2;
    }
    
    message OnUpdateRequest {
      string typeinstance_id = 1;
      uint32 new_resource_version = 2;
      bytes new_value = 3;
      optional bytes context = 4;
    }
    
    message OnUpdateResponse {
      optional bytes context = 1;
    }
    
    message OnDeleteRequest {
      string typeinstance_id = 1;
      bytes context = 2;
    }
    
    message OnDeleteResponse {}
    
    message GetValueRequest {
      string typeinstance_id = 1;
      uint32 resource_version = 2;
      bytes context = 3;
    }
    
    message GetValueResponse {
      optional bytes value = 1;
    }
    
    
    // lock messages
    
    message GetLockedByRequest {
      string typeinstance_id = 1;
      bytes context = 2;
    }
    
    message GetLockedByResponse {
      optional string locked_by = 1;
    }
    
    message OnLockRequest {
      string typeinstance_id = 1;
      bytes context = 2;
      string locked_by = 3;
    }
    
    message OnLockResponse {}
    
    message OnUnlockRequest {
      string typeinstance_id = 1;
      bytes context = 2;
    }
    
    message OnUnlockResponse {}
    
    // services
    
    service StorageBackend {
      // value
      rpc GetValue(GetValueRequest) returns (GetValueResponse);
      rpc OnCreate(OnCreateRequest) returns (OnCreateResponse);
      rpc OnUpdate(OnUpdateRequest) returns (OnUpdateResponse);
      rpc OnDelete(OnDeleteRequest) returns (OnDeleteResponse);
    
      // lock
      rpc GetLockedBy(GetLockedByRequest) returns (GetLockedByResponse);
      rpc OnLock(OnLockRequest) returns (OnLockRequest);
      rpc OnUnlock(OnUnlockRequest) returns (OnUnlockRequest);
    }
    ```

    </details>

    An implementation of such service may vary between two use cases:

    1. CRUD operations on output TypeInstance actually manages external resource (e.g. Vault) -> onCreate, onUpdate, and onDelete actually creates, updates and deletes a given resource.
    1. output TypeInstance represents external resources managed in different way (e.g. via Capact actions - like Helm Runner). IMO we shouldn't move actual Helm release installation to TypeInstance "constructor").

        - The service can also implement watch for external resources (e.g. Kubernetes secrets) and call `createTypeInstances` and `deleteTypeInstances` Hub mutations. We may provide Go framework to speed up such development, similarly as we have with Runner concept.

1. The service is installed inside the cluster. As the service manages potentially sensitive data, the access for such service is limited.

    - It doesn't expose any publicly available API. For any communication between Hub and the service, the in-cluster connection is used.
    - A proper Kubernetes [Network policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/) configuration need to be applied to limit access to the service.
    - The service is deployed in restricted namespace, where user cannot have access to. The namespace `capact-system` can be used for such deployment, as an override for Helm runner, which currently takes the namespace from the Platform context.
    - As already mentioned, the recommended way to deploy such service is to use Capact manifests. The access to run such Implementations may be restricted to Cluster Admin only with Hub authorization. Such Hub authorization will be implemented as a part of [User authentication and authorization](https://github.com/capactio/capact/issues/508) epic.

1. The service could be implemented using one of the following solutions, or other alternatives:

    - [Dapr secrets](https://docs.dapr.io/developing-applications/building-blocks/secrets/secrets-overview/)
    - [Kubernetes external secrets](https://github.com/external-secrets/kubernetes-external-secrets)
    - [vault-k8s](https://github.com/hashicorp/vault-k8s)
    - [db](https://upper.io/v4/getting-started/)
    - [go-cloud](https://github.com/google/go-cloud)
    - [stow](https://github.com/graymeta/stow)

## Configuring storage backends

The storage backends configuration consists of two different parts: default backend storage for TypeInstances with static values, and common TypeInstance injection.

1. System Administrator configures default backend in Policy for TypeInstances with static values.

    ```yaml
    interface:
      rules: [...] # rules for Interfaces, now nested under `interface`

    # ...

    typeInstance:
      rules:
        - typeRef:
            path: "cap.type.aws.auth.credentials"
            revision: "0.1.0" # optional
          backend:
            id: "00fd161c-01bd-47a6-9872-47490e11f996" # Vault backend storage
        - typeRef:
            path: "cap.type.aws.*" # for any Type reference starting with such prefix
          backend:
            id: "31bb8355-10d7-49ce-a739-4554d8a40b63" # AWS secrets manager
        - typeRef:
            path: "cap.*" # Any other Type reference
          backend:
            id: a36ed738-dfe7-45ec-acd1-8e44e8db893b
            description: "Default Capact PostgreSQL backend"
    ```

    Default storage backend for any Type reference (`cap.*`) should have `context` empty (`null`) or optional, in order to work properly. When updating Policy, the default backend storage will be validated to see if it meets the criteria.

1. To make it easy to define injection of backend storages for TypeInstances with dynamic values, in Global and Action Policies we introduce additional feature - common TypeInstance injection configuration:

    ```yaml
    interface:
      rules: [] # rules for Interfaces, now nested under `interface`

      default: # properties applied to all rules above
        inject:
          requiredTypeInstances:
            - id: "3ef2e4ac-9070-4093-a3ce-142139fd4a16"
              description: "Helm storage (cap.type.helm.storage:0.1.0)"

              # it works globally, not only for storage backends:
            - id: "c4f66896-c2a1-4031-b847-b55ae6a26c80"
              description: "GCP SA (cap.type.gcp.auth.service-account:0.1.0)"

    typeInstance:
      rules: [] # ...
    ```

    Such list of default TypeInstance to inject will be automatically merged with other injection rules for all Interface rules. The required TypeInstances are injected into workflow only if such TypeInstance is truly required ([see source code](https://github.com/capactio/capact/blob/48502d764aaab9fd4133c2ad70ba82501ad7a35b/pkg/hub/client/policy_enforced_client.go#L123)).

1. Of course, System Administrator or System User may override the defaults using current Policy syntax:

    ```yaml
    interface:
      rules:
        - interface:
            path: cap.interface.database.postgresql.install 
          oneOf:
            - implementationConstraints:
                  # constraints to select Bitnami PostgreSQL installation, for example:
                  path: cap.implementation.bitnami.postgresql.install
              inject:
                requiredTypeInstances:
                  - id: b4cf15d2-79b1-45ee-9729-6b83289ecabc # Different TypeInstance of `cap.type.helm.storage` Type - it will be used instead of the one from `interface.rules.default.inject`
                    description: "Helm Release storage"         

      default: # properties applied to all rules above
        inject:
          requiredTypeInstances:
          - id: "3ef2e4ac-9070-4093-a3ce-142139fd4a16"
            description: "Helm storage (cap.type.helm.storage:0.1.0)"
    ```

## Uninstalling storage backends

As described in the [Workflow syntax - Create](#workflow-syntax---create) section, every TypeInstance that uses a given storage backend, will use the `uses` property set:

```yaml
usesRelations: # automatically create relation between TypeInstance using a given backend
  - from: helm-release
    to: 3ef2e4ac-9070-4093-a3ce-142139fd4a16 # Helm storage backend
```

In that way, a given storage backend will contain `usedBy` relations.

According to the accepted [Rollback](./20201209-action-rollback.md) proposal:
- User won't be able to delete TypeInstance manually, but will run Rollback procedure instead.
- A given TypeInstance which contain any `usedBy` reference, cannot be deleted unless all related TypeInstances are deleted.

In other words, we will prevent removal of any storage backend that is used without any additional implementation from our side.

## GraphQL API

The new GraphQL API can be used both on CLI and UI.

### List storage backends

To list all available StorageBackends in Hub:

```graphql
query {
   types(filter: { pathPattern: "cap.core.hub.storage.*" }) {
        name
        prefix
        path
    }
}
```

To list all configured StorageBackends in Capact:

```graphql
# Ideally, but it could be too complicated:
query ListTypeInstancesWithTypeRefFilter {
  typeInstances(
    filter: { typeRef: { path: "cap.core.hub.storage.*" } } # queries public Hub to fetch all Types attached to `cap.core.hub.storage` and return all TypeInstances which are of one of these TypeRefs
  ) {
    ...TypeInstance
  }
}

# Alternatively: introduce `cap.core.attribute.hub.storage.backend` Attribute and simply do:

query ListTypeInstancesWithAttributesAndTypeRefFilter {
  typeInstances(
    filter: {
      attributes: [
        { pathPattern: "cap.core.attribute.hub.storage.backend", rule: INCLUDE }
      ]
    }
  ) {
    ...TypeInstance
  }
}
```

### Get TypeInstance details

To see the value for all TypeInstances, we can use the following query:

```graphql
query ListTypeInstances {
  typeInstances {
    id
    typeRef {
      path
      revision
    }
    lockedBy # resolver which calls proper storage backend service to ask for lock status
    latestResourceVersion {
      resourceVersion
      createdBy
      metadata {
        attributes {
          path
          revision
        }
      }
      spec {
        value # resolver which calls proper storage backend service to ask for a given ResourceVersion value
      }
      backend { # new property
        context
      }
    }
    backend { # new property
      # Initially, we can return only TypeInstance ID
      """TypeInstance ID"""
      id

      # Later, we can resolve full details here based on the ID
      latestResourceVersion {
        metadata {
          alias # new field
        }
        value # url + contextSchema
      }
      
    }
  }
}
```

### TypeInstance create

```graphql
input CreateTypeInstanceBackendInput { # New input
  id: ID # storage backend TypeInstance ID. Optional, as it will fallback to default one if not provided

  context: Any # Properties which will be populated into the first Resource Version of the newly created TypeInstance
}

input CreateTypeInstanceInput {
  # (...)
  alias: String
  value: Any

  backend: CreateTypeInstanceBackendInput # new property
}

input CreateTypeInstancesInput {
  typeInstances: [CreateTypeInstanceInput!]!
  usesRelations: [TypeInstanceUsesRelationInput!]!
}

type Mutation {
  createTypeInstances(
      in: CreateTypeInstancesInput!
    ): [CreateTypeInstanceOutput!]!
}
```

### TypeInstance update

To properly handle TypeInstance update, the following additions to the API need to be made:

```graphql
input UpdateTypeInstanceBackendInput { # New input
  context: Any
}

"""
At least one property needs to be specified.
"""
input UpdateTypeInstanceInput {
  # (...)
  value: Any

  backend: UpdateTypeInstanceBackendInput # New property
}

input UpdateTypeInstancesInput {
  # ...

  id: ID!
  typeInstance: UpdateTypeInstanceInput!
}

type Mutation {
  updateTypeInstances(in: [UpdateTypeInstancesInput]!): [TypeInstance!]!
}
```

## Dynamic TypeInstance projections

### Introduction

In Capact manifests, there is another common pattern. Apart from TypeInstances describing external resources, there are TypeInstances which unify output based on related TypeInstances.

For example:
- The [`cap.interface.database.postgresql.install`](https://github.com/capactio/hub-manifests/blob/1eab022b7960acb92e018103a38d7852b30c2460/manifests/interface/database/postgresql/install.yaml) Interface produces [PostgreSQL Config](https://github.com/capactio/hub-manifests/blob/1eab022b7960acb92e018103a38d7852b30c2460/manifests/type/database/postgresql/config.yaml) TypeInstance, which contains data from a Helm release or AWS RDS for PostgreSQL Instance.
- The [`cap.interface.productivity.mattermost.install`](https://github.com/capactio/hub-manifests/blob/1eab022b7960acb92e018103a38d7852b30c2460/manifests/interface/productivity/mattermost/install.yaml) Interface produces [Mattermost Config](https://github.com/capactio/hub-manifests/blob/1eab022b7960acb92e018103a38d7852b30c2460/manifests/type/productivity/mattermost/config.yaml) TypeInstance, which contains data from Mattermost Helm release.

There could be some cases where the output TypeInstance represents data gathered from multiple related TypeInstances. For example, a Kubernetes cluster TypeInstance could contain data from Terraform TypeInstance, and also a separate Kubeconfig TypeInstance.

To avoid implementing a special storage backend service every time we have such case, we introduce a dedicated storage backend for convenience.

### Go Template backend storage

1. The following manifests are needed to register such backend storage:

    <details> <summary><code>cap.core.type.hub.storage.gotemplate</code> Type</summary>

    ```yaml
    ocfVersion: 0.0.1
    revision: 0.0.1
    kind: Type
    metadata:
      path: cap.core.type.hub.storage.gotemplate
    spec:
      additionalRefs: [] # additionalRef not needed if it resides under `cap.core.type.hub.storage`
      jsonSchema:
        value: # JSON schema with:
          {
          "$schema": "http://json-schema.org/draft-07/schema",
          "type": "object",
          "title": "The root schema",
          "required": [
            "url",
            "contextSchema"
          ],
          "properties": {
            "url": { # url of hosted app, which implements storage backend ProtocolBuffers interface.
              "$id": "#/properties/url",
              "type": "string",
              "format": "uri"
            },
            "contextSchema": { # JSON schema which describes additional properties passed in Capact workflow
              "const": { # see http://json-schema.org/draft/2019-09/json-schema-validation.html#rfc.section.6.1.3
                {
                  "$schema": "http://json-schema.org/draft-07/schema",
                  "type": "object",
                  "default": {},
                  "examples": [
                    {
                      "template" "username: '{{ .username }}'\npassword: '{{ .password }}'"
                      "variables": {
                        "foo": {
                          "relatedTypeInstanceAlias": "alias"
                        },
                        "bar": {
                          "typeInstanceID": "uuid"
                        }
                      }
                    }
                  ],
                  "required": [
                    "variables",
                    "template"
                  ],
                  "properties": {
                    "template": {
                    "$id": "#/properties/template",
                      "type": "string"
                    },
                    "variables": {
                      "$id": "#/properties/variables",
                      "type": "object",
                      "additionalProperties": false,
                      "patternProperties": {
                        "^[a-zA-Z0-9]*$": {
                          "type": "object",
                          "properties": {
                              "typeInstanceID": {
                                "type": "string"
                              },
                              "relatedTypeInstanceAlias": {
                                "type": "string"
                              }
                          }
                        }
                      }
                    }
                  },
                  "additionalProperties": true
                }        
              }
            },
            "acceptValue": { # specifies if a given storage backend (app) accepts TypeInstance value while creating/updating TypeInstance, or just context.
              "$id": "#/properties/acceptValue",
              "type": "boolean",
              "const": false # in this case - no
            },
          },
          "additionalProperties": false
        }
    ```

    </details>

    <details> <summary><code>cap.core.type.hub.storage.gotemplate</code> TypeInstance</summary>

    ```yaml
    id: abd48b8c-99bd-40a7-99c0-047bd69f1db8
    typeRef:
        path: cap.core.type.hub.storage.gotemplate
        revision: 0.1.0
    latestResourceVersion:
      metadata:
        alias: capact-gotemplate
        attributes:
        - path: cap.core.attribute.hub.storage.backend
          revision: 0.1.0
      value:
        url: "storagebackend-gotemplate.capact-system:50051"
        acceptValue: false # If it is `true`, the value is validated by Hub whether it matches the JSON schema of the Type from typeRef of the TypeInstance
        contextSchema: |-
          {
            "$schema": "http://json-schema.org/draft-07/schema",
            "type": "object",
            "default": {},
            "examples": [
              {
                "template" "username: '{{ .username }}'\npassword: '{{ .password }}'"
                "variables": {
                  "foo": {
                    "relatedTypeInstanceAlias": "alias"
                  },
                  "bar": {
                    "typeInstanceID": "uuid"
                  }
                }
              }
            ],
            "required": [
              "variables",
              "template"
            ],
            "properties": {
              "template": {
              "$id": "#/properties/template",
                "type": "string"
              },
              "variables": {
                "$id": "#/properties/variables",
                "type": "object",
                "additionalProperties": false,
                "patternProperties": {
                  "^[a-zA-Z0-9]*$": {
                    "type": "object",
                    "properties": {
                      "typeInstanceID": {
                        "type": "string"
                      },
                      "relatedTypeInstanceAlias": {
                        "type": "string"
                      }
                    }
                  }
                }
              }
            },
            "additionalProperties": true
          }
    backend: 
      id: a36ed738-dfe7-45ec-acd1-8e44e8db893b # PostgreSQL backend
    ```

    </details>

1. Content Developer can define an Argo artifact:

    ```yaml
    backend:
      context:
        template: |-
          host: "{{ index .mattermost.values.ingress.hosts 0 }}"
          version: "{{ .mattermost.values.image.tag }}"
        variables:
          mattermost:
            # Reference by `metadata.alias` of the TypeInstance
            # The TypeInstance must be in the `uses` tree
            # Based on the alias the storage backend will resolve ID and return it
            relatedTypeInstanceAlias: mattermost-config
    ```

1. During TypeInstance upload, the Go Template storage backend service will resolve the TypeInstance ID based on the `relatedTypeInstanceAlias`, by traversing the `uses` relation tree. It also validates whether the template rendering succeeds based on provided TypeInstances. If so, it returns mutated `context`:

    ```yaml
    context:
        template: |-
          host: "{{ index .mattermost.values.ingress.hosts 0 }}"
          version: "{{ .mattermost.values.image.tag }}"
        variables:
          mattermost:
            # relatedTypeInstanceAlias removed as it is not needed anymore
            typeInstanceID: b895d2d4-d0e0-4f7c-9666-4c3d197d1795 # resolved ID based on `relatedTypeInstanceAlias`. It will be used for further template rendering
    ```

1. So, in a result, the final TypeInstance produced will look like this:

    ```yaml
    id: b895d2d4-d0e0-4f7c-9666-4c3d197d1795
    typeRef:
      path: cap.type.productivity.mattermost.config
      revision: 0.1.0
    latestResourceVersion:
      resourceVersion: 1
      backend:
        context: # additional parameters that might be modified via the service handling `onCreate` hook
          template: |-
            host: "{{ index mattermost.values.ingress.hosts 0 }}"
            version: "{{ mattermost.values.image.tag }}"
          variables:
            mattermost:
              relatedTypeInstanceAlias: mattermost-config # kept for better readability
              typeInstanceID: b895d2d4-d0e0-4f7c-9666-4c3d197d1795 # resolved ID based on `relatedTypeInstanceAlias`. It will be used for further template rendering
    backend: 
      id: abd48b8c-99bd-40a7-99c0-047bd69f1db8 # capact-gotemplate backend - resolved UUID
    ```

1. When fetching the value of such TypeInstance, Go Template storage backend renders the template based on up-to-date values and returns:

    ```yaml
    host: mattermost.capact.local
    version: 6.2.0
    ```

For more details, see the [Workflow syntax - Create](#workflow-syntax---create) paragraph.

### Helm runner templating

Sometimes we used Helm template rendering for some outputs, with usage of some Helm chart templating functions from `_helpers.tpl`:

```yaml
output:
  goTemplate: |
    host: '{{ template "common.names.fullname" . }}.{{ .Release.Namespace }}'
    port: '{{ template "postgresql.port" . }}'
    defaultDBName: '{{ template "postgresql.database" . }}'
    superuser:
      # It cannot be changed
      username: 'postgres'
      password: '{{ template "postgresql.password" . }}'
```

Unfortunately, that won't be possible anymore, and instead we should get all the values from Helm release details or additional external resources, like Kubernetes ConfigMaps or Secrets. In that case, Content Developer outputs additional TypeInstances and use them as template variables.

## Rejected ideas

### Registering storage backends

1. Enforcing convention of JSON schema in storage backend Type.

    As described in proposal, every storage backend Type should follow the convention of having JSON schema with `uri` and `contextSchema` properties.

    That could be possible if we implement an ability to define validating JSON schema for Type nodes (e.g. `cap.core.type.hub.storage`), and use such schemas to validate Types attached to these nodes (via `spec.additionalRefs` property). For example, the `cap.core.hub.storage` node could have JSON Schema defined, which validates Type values (JSON schema) attached to such node. In the end, that would be JSON schema validating another JSON schema.
    
    **Reason:** It is possible, but it's complex and brings too little benefits for now to implement it.

1. Adding optional `TypeInstance.metadata.name` or `alias`, which is unique across all TypeInstances and immutable regardless resourceVersion. It would allow easier referencing storage backends in the `TypeInstance.spec.backend` field:

    ```yaml
    id: 3ef2e4ac-9070-4093-a3ce-142139fd4a16
    metadata:
      name: helm-storage
    typeRef:
      path: cap.type.helm.storage
      revision: 0.1.0
    latestResourceVersion:
       #...
    backend: capact-postgresql # immutable - contains TypeInstance ID or unique alias
      # if not provided, fallback to default one during TypeInstance creation
    ```

    **Reason:** It is not really needed as we can use unique IDs to reference such backends. Also, we can expose GraphQL API which resolves details of a given storage backend based on the ID.

1. Dedicated entity of StorageBackend

    Such resource could reside in Local Hub, but it wouldn't be an OCF manifest. System Administrator should be able to manage them via GraphQL API, CLI and UI.

    **Reasons:**
    - We would still need some kind of StorageBackend templates (with `contextSchema` JSON schema) in public Hub
    - How we would be able to output such as a result of an Action? It could be done in a hacky way to output it as a side effect of running Action (not explicitly), but that would be definitely not elegant
    - We would need to have additional API

### Workflow syntax

1. Keep the Argo artifact value as it is, and add additional syntax:

    ```yaml
    # default - static
    capact-outputTypeInstances:
      - name: mattermost-config # still static
        from: additional
        # no backend definition -> used default (default storage backend (TypeInstance) is annotated with `cap.core.attribute.default`)

    # option 2 - create TypeInstance on external storage
    capact-outputTypeInstances:
      - name: helm-release
        from: helm-release # values
        backend: vault
        context: "{{steps.foo.output.artifacts.foo}}" 

    # option 3 - register something which already exist as external TypeInstance - based on context
    capact-outputTypeInstances:
      - name: helm-release
        backend: helm-storage
        context: "{{steps.foo.output.artifacts.foo}}"
    ```

    **Reason:** More complex usage in the workflow, and more complex implementation as well.

### Storage backend service implementation

1. Using Actions as a way to do CRUD operations (separate Interface/Implementation per Create/Update/Get/Delete operation)
 
    **Reason:** While the idea may seem exciting, that would be really time consuming and ineffective. We are too far from the point at where we can think about such solution. 

### Configuring default storage backends

1. Using dedicated Attribute to specify which storage backend TypeInstance should be selected by default

    **Reason**: We still need to configure Policy to configure common TypeInstance injection.

1. Using the `cap.*` rule to define common TypeInstance injection

    **Reason**: That would be too difficult to understand for System Administrator and System User. Additional property seem as better solution.

## Potential future evolution

There is another concept, where storage backend service handles both CRUD operations for TypeInstances and external resources. For example, Helm backend service would manage Helm Releases. Underneath it could use Helm operator.

### Key points

- Get rid of current runners and rework them to asynchronous services which manage resources
- Extend the gRPC + Protocol Buffers API and return status of async operation
- Still, there should be a container which passes inputs to the async service and waits for completion:
  - Periodically polling status of the async task from the service.
  - We need runner-specific Capact manifests to have proper Interface with Input/Output Types (for example, separate manifests for installing Helm charts, Terraform apply etc.)
  - We can have one generic Docker image.
- To upload the TypeInstance into Hub:
  - The service could upload TypeInstance directly and pass its ID into workflow.
  - For static values: use dedicated Action in workflow?
    - It might be related to the issue [Prepare PoC for generic upload of TypeInstances even if Action failed](https://github.com/capactio/capact/issues/563).
  - At the end of the workflow we could just set relations for existing TypeInstances

### Summary

Initially, I backed off from this idea as it seemed to me as too invasive. Depending how far we want to go with it, it could bring additional problems to solve, apart from the one described within this proposal. Fortunately, most of the solutions described in this document would be also applicable to the future evolution of the concept.

The proposal brings many changes already, so it might be beneficial to consider it as a part of investigation or spike before actual implementation of Delegated storage.

## Consequences

Once approved, we need to address the following list of items:

1. Investigate our approach to implementation
    - taking into account the [Potential feature evolution](#potential-future-evolution) which may impact implementation
1. Implement these two Type features:
    - [Additional references to parent nodes](https://capact.io/docs/feature/type-features/#additional-references-to-parent-nodes)
    - [Find Types based on prefix of parent nodes](https://capact.io/docs/feature/type-features#find-types-based-on-prefix-of-parent-nodes)
1. [Add TypeInstance `alias` metadata field](https://github.com/capactio/capact/issues/579)
    - Optional until we want to implement the [dynamic TypeInstance projections(#dynamic-typeinstance-projections) according to the proposal
1. [Validate TypeInstance input](https://github.com/capactio/capact/issues/544)
1. Implement [User authentication and authorization](https://github.com/capactio/capact/issues/508)
    - Optional - needed only if we want to restrict Hub content access
1. Extend TypeInstance with the `TypeInstance.spec.backend` field
1. Rewrite Local Hub to Go
    - Make Local Hub to use PostgreSQL
    - Extend GraphQL API
1. Implement Go Template storage backend services:
    - default storage backend service for PostgreSQL
    - service for TypeInstance projection
    - At the very beginning it can be simplified to built-in database driver in Local Hub. Later we can implement the gRPC+ProtoBuf external service concept
1. Adjust all Implementations to use latest workflow syntax (Argo workflows artifact content):
    - Use dynamic TypeInstance projection for output TypeInstances
    - We can try to keep backward compatibility if needed and for older manifests still treat whole Argo artifacts as TypeInstance value
    - In case of backward compatibility we can adjust just a few manifests for showcase purposes
1. Handle multiple backends for the TypeInstance upload from workflow
    - Extend `capact-outputTypeInstances` syntax
    - Set proper `uses` relations between storage backend TypeInstance and other TypeInstances
    - Modify TypeInstance create/update/delete images (named as "Argo actions") to take new input
1. Update Policy
    - Add new properties
    - Handle common TypeInstance injections
1. Update documentation 
    - Policy
    - Content Development guide
    - Type features
1. Runners
    - Remove `output.goTemplate`
    - Stop supporting usage of funcs from `_helpers.tpl` in case of Helm runner
