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
// Direct grant relations are prefixed with "granted_" to avoid name conflicts
// with permit functions (Keto skips permit evaluation when a relation shares
// the same name as a permit function).
// The "service" relation bridges service bots from tenancy_access.
class service_trustage implements Namespace {
  related: {
    owner: profile_user[]
    admin: profile_user[]
    member: profile_user[]
    service: (profile_user | tenancy_access)[]

    // Default app permissions (direct grants)
    granted_event_ingest: (profile_user | service_trustage)[]
    granted_workflow_manage: (profile_user | service_trustage)[]
    granted_workflow_view: (profile_user | service_trustage)[]
    granted_instance_view: (profile_user | service_trustage)[]
    granted_instance_retry: (profile_user | service_trustage)[]
    granted_execution_view: (profile_user | service_trustage)[]
    granted_execution_retry: (profile_user | service_trustage)[]

    // Formstore app permissions (direct grants)
    granted_form_definition_manage: (profile_user | service_trustage)[]
    granted_form_definition_view: (profile_user | service_trustage)[]
    granted_form_submit: (profile_user | service_trustage)[]
    granted_submission_view: (profile_user | service_trustage)[]
    granted_submission_update: (profile_user | service_trustage)[]
    granted_submission_delete: (profile_user | service_trustage)[]

    // Queue app permissions (direct grants)
    granted_queue_manage: (profile_user | service_trustage)[]
    granted_queue_view: (profile_user | service_trustage)[]
    granted_item_enqueue: (profile_user | service_trustage)[]
    granted_queue_item_view: (profile_user | service_trustage)[]
    granted_counter_manage: (profile_user | service_trustage)[]
    granted_stats_view: (profile_user | service_trustage)[]
  }

  permits = {
    // Default app permits — admin-level
    event_ingest: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.granted_event_ingest.includes(ctx.subject),
    workflow_manage: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.granted_workflow_manage.includes(ctx.subject),
    workflow_view: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject) ||
      this.related.granted_workflow_view.includes(ctx.subject),
    instance_view: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject) ||
      this.related.granted_instance_view.includes(ctx.subject),
    instance_retry: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.granted_instance_retry.includes(ctx.subject),
    execution_view: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject) ||
      this.related.granted_execution_view.includes(ctx.subject),
    execution_retry: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.granted_execution_retry.includes(ctx.subject),

    // Formstore app permits — admin-level for manage/update/delete, view-level for view/submit
    form_definition_manage: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.granted_form_definition_manage.includes(ctx.subject),
    form_definition_view: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject) ||
      this.related.granted_form_definition_view.includes(ctx.subject),
    form_submit: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject) ||
      this.related.granted_form_submit.includes(ctx.subject),
    submission_view: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject) ||
      this.related.granted_submission_view.includes(ctx.subject),
    submission_update: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.granted_submission_update.includes(ctx.subject),
    submission_delete: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.granted_submission_delete.includes(ctx.subject),

    // Queue app permits — admin-level for manage, view-level for view/enqueue
    queue_manage: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.granted_queue_manage.includes(ctx.subject),
    queue_view: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject) ||
      this.related.granted_queue_view.includes(ctx.subject),
    item_enqueue: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject) ||
      this.related.granted_item_enqueue.includes(ctx.subject),
    queue_item_view: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject) ||
      this.related.granted_queue_item_view.includes(ctx.subject),
    counter_manage: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.granted_counter_manage.includes(ctx.subject),
    stats_view: (ctx: Context): boolean =>
      this.related.service.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject) ||
      this.related.granted_stats_view.includes(ctx.subject),
  }
}

export { profile_user, tenancy_access, service_trustage }
