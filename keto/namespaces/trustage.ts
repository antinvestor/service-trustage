// Keto Namespace Configuration for Trustage
// Using Ory Permission Language (OPL) - TypeScript-like DSL
//
// Two-layer authorization:
//   Layer 1 — tenancy_access: Data access gate (can this caller access this partition?)
//   Layer 2 — service_trustage: Functional permissions per tenant/partition

import { Namespace, Context } from "@ory/keto-namespace-types"

// profile_user is the platform-wide user identity namespace, shared across all services.
class profile_user implements Namespace {}

// tenancy_access gates data access per tenant/partition (Layer 1).
// "member" = regular user, "service" = service bot (system_internal role).
class tenancy_access implements Namespace {
  related: {
    member: profile_user[]
    service: profile_user[]
  }
}

// service_trustage holds functional permission tuples for the trustage service.
// Permissions are materialized as direct tuples (one per user per permission).
// The "service" relation bridges service bots from tenancy_access.
class service_trustage implements Namespace {
  related: {
    owner: profile_user[]
    admin: profile_user[]
    member: profile_user[]
    service: (profile_user | tenancy_access)[]

    // Default app permissions
    ingest_event: (profile_user | service_trustage)[]
    manage_workflow: (profile_user | service_trustage)[]
    view_workflow: (profile_user | service_trustage)[]
    view_instance: (profile_user | service_trustage)[]
    retry_instance: (profile_user | service_trustage)[]
    view_execution: (profile_user | service_trustage)[]
    retry_execution: (profile_user | service_trustage)[]

    // Formstore app permissions
    manage_form_definition: (profile_user | service_trustage)[]
    view_form_definition: (profile_user | service_trustage)[]
    submit_form: (profile_user | service_trustage)[]
    view_submission: (profile_user | service_trustage)[]
    update_submission: (profile_user | service_trustage)[]
    delete_submission: (profile_user | service_trustage)[]

    // Queue app permissions
    manage_queue: (profile_user | service_trustage)[]
    view_queue: (profile_user | service_trustage)[]
    enqueue_item: (profile_user | service_trustage)[]
    view_queue_item: (profile_user | service_trustage)[]
    manage_counter: (profile_user | service_trustage)[]
    view_stats: (profile_user | service_trustage)[]
  }
}

export { profile_user, tenancy_access, service_trustage }
