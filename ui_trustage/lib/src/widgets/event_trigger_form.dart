import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/trustage_providers.dart';
import 'trustage_panel.dart';

/// Form for triggering (ingesting) an external event.
class EventTriggerForm extends ConsumerStatefulWidget {
  const EventTriggerForm({super.key});

  @override
  ConsumerState<EventTriggerForm> createState() => _EventTriggerFormState();
}

class _EventTriggerFormState extends ConsumerState<EventTriggerForm> {
  final _eventTypeCtl = TextEditingController(text: 'order.created');
  final _sourceCtl = TextEditingController(text: 'ops-console');
  final _idempotencyCtl = TextEditingController();
  final _payloadCtl = TextEditingController(
    text: '{\n  "order_id": "ORD-001",\n  "amount": 1500,\n  "currency": "KES"\n}',
  );
  bool _busy = false;
  String? _statusMessage;

  @override
  void dispose() {
    _eventTypeCtl.dispose();
    _sourceCtl.dispose();
    _idempotencyCtl.dispose();
    _payloadCtl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return TrustagePanel(
      eyebrow: 'Events',
      title: 'Trigger an Event',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _eventTypeCtl,
                  decoration: const InputDecoration(
                    labelText: 'Event Type',
                    isDense: true,
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: TextField(
                  controller: _sourceCtl,
                  decoration: const InputDecoration(
                    labelText: 'Source',
                    isDense: true,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _idempotencyCtl,
            decoration: const InputDecoration(
              labelText: 'Idempotency Key (optional)',
              isDense: true,
            ),
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _payloadCtl,
            decoration: const InputDecoration(
              labelText: 'Payload (JSON)',
              alignLabelWithHint: true,
              isDense: true,
            ),
            maxLines: 5,
            style: theme.textTheme.bodySmall?.copyWith(fontFamily: 'monospace'),
          ),
          const SizedBox(height: 12),
          FilledButton.icon(
            onPressed: _busy ? null : _send,
            icon: _busy
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.bolt, size: 16),
            label: Text(_busy ? 'Sending...' : 'Send event'),
          ),
          if (_statusMessage != null) ...[
            const SizedBox(height: 8),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerLow,
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(_statusMessage!, style: theme.textTheme.bodySmall),
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _send() async {
    if (_eventTypeCtl.text.isEmpty) {
      setState(() => _statusMessage = 'Event type is required');
      return;
    }

    Map<String, dynamic>? payload;
    if (_payloadCtl.text.isNotEmpty) {
      try {
        payload = jsonDecode(_payloadCtl.text) as Map<String, dynamic>;
      } catch (e) {
        setState(() => _statusMessage = 'Invalid JSON: $e');
        return;
      }
    }

    setState(() {
      _busy = true;
      _statusMessage = null;
    });

    try {
      final notifier = ref.read(trustageActionProvider.notifier);
      final resp = await notifier.ingestEvent(
        eventType: _eventTypeCtl.text,
        source: _sourceCtl.text,
        idempotencyKey: _idempotencyCtl.text,
        payload: payload,
      );
      if (mounted) {
        final eventId = resp.event.eventId;
        setState(() {
          _busy = false;
          _statusMessage = resp.idempotent
              ? 'Event matched (idempotent): $eventId'
              : 'Event accepted as $eventId';
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _busy = false;
          _statusMessage = 'Error: $e';
        });
      }
    }
  }
}
