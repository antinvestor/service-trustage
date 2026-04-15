import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import 'package:antinvestor_api_runtime/antinvestor_api_runtime.dart' as rt;

import 'trustage_panel.dart';
import 'trustage_status_badge.dart';
import 'status_helpers.dart';

/// Panel showing active signal waits and pending signal messages.
class BlockedStatePanel extends StatelessWidget {
  const BlockedStatePanel({
    super.key,
    required this.signalWaits,
    required this.signalMessages,
  });

  final List<rt.SignalWait> signalWaits;
  final List<rt.SignalMessage> signalMessages;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final activeWaits = signalWaits
        .where((w) => isWaiting(w.status))
        .toList();
    final pendingMessages = signalMessages
        .where((m) => m.status.toLowerCase().contains('pending'))
        .toList();

    if (activeWaits.isEmpty && pendingMessages.isEmpty) {
      return const SizedBox.shrink();
    }

    final fmt = DateFormat.yMMMd().add_jm();

    return TrustagePanel(
      eyebrow: 'Blocked',
      title: 'Waiting States',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (activeWaits.isNotEmpty) ...[
            Text('Signal Waits', style: theme.textTheme.labelSmall?.copyWith(
              fontWeight: FontWeight.w600,
              letterSpacing: 0.5,
            )),
            const SizedBox(height: 6),
            ...activeWaits.map((w) => Container(
                  margin: const EdgeInsets.only(bottom: 6),
                  padding: const EdgeInsets.all(10),
                  decoration: BoxDecoration(
                    color: const Color(0xFFFFFBEB),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: const Color(0xFFFDE68A)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Text(
                            w.signalName,
                            style: const TextStyle(
                              fontSize: 12,
                              fontWeight: FontWeight.w600,
                              color: Color(0xFFD97706),
                            ),
                          ),
                          const Spacer(),
                          TrustageStatusBadge(status: w.status),
                        ],
                      ),
                      const SizedBox(height: 4),
                      Text(
                        'Execution: ${shortId(w.executionId)}',
                        style: theme.textTheme.labelSmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                      if (w.hasTimeoutAt())
                        Text(
                          'Timeout: ${fmt.format(w.timeoutAt.toDateTime())}',
                          style: theme.textTheme.labelSmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                    ],
                  ),
                )),
          ],
          if (pendingMessages.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text('Pending Signals', style: theme.textTheme.labelSmall?.copyWith(
              fontWeight: FontWeight.w600,
              letterSpacing: 0.5,
            )),
            const SizedBox(height: 6),
            ...pendingMessages.map((m) => Container(
                  margin: const EdgeInsets.only(bottom: 6),
                  padding: const EdgeInsets.all(10),
                  decoration: BoxDecoration(
                    color: const Color(0xFFF5F3FF),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: const Color(0xFFDDD6FE)),
                  ),
                  child: Row(
                    children: [
                      Text(
                        m.signalName,
                        style: const TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.w600,
                          color: Color(0xFF7C3AED),
                        ),
                      ),
                      const Spacer(),
                      TrustageStatusBadge(status: m.status),
                    ],
                  ),
                )),
          ],
        ],
      ),
    );
  }
}
