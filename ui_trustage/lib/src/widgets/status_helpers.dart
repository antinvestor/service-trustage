import 'package:flutter/material.dart';

/// Strips common prefixes and humanizes a status token.
/// e.g. "EXECUTION_STATUS_RUNNING" → "running"
String humanizeStatus(String raw) {
  var s = raw;
  for (final prefix in [
    'WORKFLOW_STATUS_',
    'INSTANCE_STATUS_',
    'EXECUTION_STATUS_',
    'STATUS_',
  ]) {
    if (s.startsWith(prefix)) {
      s = s.substring(prefix.length);
      break;
    }
  }
  return s.replaceAll('_', ' ').toLowerCase();
}

/// Returns a color for the given status string.
Color statusColor(String status) {
  final s = status.toLowerCase();
  if (s.contains('fatal') || s.contains('failed') || s.contains('invalid')) {
    return const Color(0xFFE11D48); // rose-600
  }
  if (s.contains('waiting') || s.contains('suspended')) {
    return const Color(0xFFD97706); // amber-600
  }
  if (s.contains('running') || s.contains('dispatched') || s.contains('pending')) {
    return const Color(0xFF0284C7); // sky-600
  }
  if (s.contains('retry')) {
    return const Color(0xFFEA580C); // orange-600
  }
  if (s.contains('completed') || s.contains('active')) {
    return const Color(0xFF059669); // emerald-600
  }
  if (s.contains('timed_out') || s.contains('timeout')) {
    return const Color(0xFF92400E); // amber-800
  }
  if (s.contains('stale') || s.contains('cancelled') || s.contains('archived')) {
    return const Color(0xFF57534E); // stone-600
  }
  return const Color(0xFF78716C); // stone-500
}

/// Returns a light background color for the given status.
Color statusBackground(String status) {
  return statusColor(status).withAlpha(20);
}

/// Returns true if the execution status can be retried.
bool canRetry(String status) {
  final s = status.toLowerCase();
  return s.contains('failed') ||
      s.contains('fatal') ||
      s.contains('timed_out') ||
      s.contains('timeout') ||
      s.contains('invalid') ||
      s.contains('retry');
}

/// Returns true if the execution is in a waiting state.
bool isWaiting(String status) {
  return status.toLowerCase().contains('waiting');
}

/// Shortens a UUID to its first 8 characters.
String shortId(String id) {
  if (id.length <= 8) return id;
  return id.substring(0, 8);
}
