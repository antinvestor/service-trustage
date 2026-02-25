// Keto Namespace Configuration for Trustage
// Using Ory Permission Language (OPL) - TypeScript-like DSL

import { Namespace, Context } from "@ory/keto-namespace-types"

// trustage_profile namespace represents users/actors
class profile implements Namespace {
  related: {
    self: profile[]
  }
}

// trustage_tenant namespace represents a tenant boundary
class trustage_tenant implements Namespace {
  related: {
    owner: profile[]
    admin: profile[]
    member: profile[]
  }

  permits = {
    ingest_event: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject),

    manage_workflow: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject),

    view_workflow: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject),

    view_instance: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject),

    retry_instance: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject),

    view_execution: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject),

    retry_execution: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject),
  }
}

export { profile, trustage_tenant }
