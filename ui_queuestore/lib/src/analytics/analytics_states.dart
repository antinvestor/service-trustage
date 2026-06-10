import 'package:antinvestor_ui_core/antinvestor_ui_core.dart';
import 'package:flutter/material.dart';

/// Maps analytics gate failures to friendly, user-facing messages.
///
/// The gate rejects malformed or disallowed queries with 400, tenant-scope
/// problems with 401/403, and signals backend outages with 5xx; none of
/// those should surface as raw exception strings.
String analyticsFailureMessage(Object error) {
  if (error is AnalyticsQueryException) {
    final status = error.statusCode;
    if (status == 400) {
      return 'This metric is not supported by the analytics service yet.';
    }
    if (status == 401 || status == 403) {
      return 'You do not have access to analytics for this tenant.';
    }
    if (status >= 500) {
      return 'Analytics is temporarily unavailable. Please try again '
          'shortly.';
    }
    return 'The analytics request failed (HTTP $status).';
  }
  return 'Could not load analytics data.';
}

/// Friendly inline error state for a failed analytics query.
class AnalyticsErrorCard extends StatelessWidget {
  const AnalyticsErrorCard({super.key, required this.error, this.onRetry});

  final Object error;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: cs.errorContainer.withValues(alpha: 0.25),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Icon(Icons.cloud_off_outlined, color: cs.error, size: 20),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              analyticsFailureMessage(error),
              style: theme.textTheme.bodySmall?.copyWith(color: cs.error),
            ),
          ),
          if (onRetry != null)
            TextButton(onPressed: onRetry, child: const Text('Retry')),
        ],
      ),
    );
  }
}
