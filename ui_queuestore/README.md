# antinvestor_ui_queuestore

Queue management UI for the trustage queuestore service: queue dashboard
(per-queue snapshot stats from the queue API) plus a thesa-gated analytics
activity section driven by the `queue.*` business metrics
(enqueue/dequeue/complete/cancel/noshow rates, operation latency,
capacity rejections). Consumes `antinvestor_ui_core`'s
`ThesaAnalyticsDataSource`; tenant scoping is enforced server-side.
