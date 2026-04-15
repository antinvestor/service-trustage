import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:antinvestor_api_runtime/antinvestor_api_runtime.dart' as rt;

import '../providers/trustage_providers.dart';
import 'status_helpers.dart';
import 'trustage_panel.dart';

/// Operator action center: retry execution/instance, send signal, resume.
class ActionCenter extends ConsumerStatefulWidget {
  const ActionCenter({
    super.key,
    required this.instance,
    this.selectedExecution,
    this.signalWaits = const [],
  });

  final rt.WorkflowInstance instance;
  final rt.WorkflowExecution? selectedExecution;
  final List<rt.SignalWait> signalWaits;

  @override
  ConsumerState<ActionCenter> createState() => _ActionCenterState();
}

class _ActionCenterState extends ConsumerState<ActionCenter> {
  final _signalNameCtl = TextEditingController();
  final _signalPayloadCtl = TextEditingController(text: '{\n  "approved": true\n}');
  String? _statusMessage;
  bool _busy = false;

  @override
  void didUpdateWidget(ActionCenter oldWidget) {
    super.didUpdateWidget(oldWidget);
    // Auto-fill signal name from first waiting signal.
    final waits = widget.signalWaits
        .where((w) => w.status.toLowerCase().contains('waiting'))
        .toList();
    if (waits.isNotEmpty && _signalNameCtl.text.isEmpty) {
      _signalNameCtl.text = waits.first.signalName;
    }
  }

  @override
  void dispose() {
    _signalNameCtl.dispose();
    _signalPayloadCtl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final exec = widget.selectedExecution;
    final inst = widget.instance;

    return TrustagePanel(
      eyebrow: 'Operator',
      title: 'Action Center',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Retry execution
          if (exec != null && canRetry(exec.status.name)) ...[
            FilledButton.icon(
              onPressed: _busy ? null : () => _retryExecution(exec.id),
              icon: const Icon(Icons.replay, size: 16),
              label: Text(
                _busy ? 'Retrying...' : 'Retry execution ${shortId(exec.id)}',
              ),
            ),
            const SizedBox(height: 8),
          ],

          // Retry instance
          if (canRetry(inst.status.name)) ...[
            OutlinedButton.icon(
              onPressed: _busy ? null : () => _retryInstance(inst.id),
              icon: const Icon(Icons.refresh, size: 16),
              label: const Text('Retry selected instance'),
            ),
            const SizedBox(height: 16),
          ],

          // Signal delivery section
          const Divider(),
          const SizedBox(height: 8),
          Text('Send Signal', style: theme.textTheme.titleSmall),
          const SizedBox(height: 8),
          TextField(
            controller: _signalNameCtl,
            decoration: const InputDecoration(
              labelText: 'Signal Name',
              hintText: 'e.g., approval_response',
              isDense: true,
            ),
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _signalPayloadCtl,
            decoration: const InputDecoration(
              labelText: 'Payload (JSON)',
              alignLabelWithHint: true,
              isDense: true,
            ),
            maxLines: 4,
            style: theme.textTheme.bodySmall?.copyWith(fontFamily: 'monospace'),
          ),
          const SizedBox(height: 8),
          FilledButton.tonalIcon(
            onPressed: _busy ? null : _sendSignal,
            icon: const Icon(Icons.send, size: 16),
            label: Text(
              'Send signal to ${shortId(inst.id)}',
            ),
          ),

          // Status message
          if (_statusMessage != null) ...[
            const SizedBox(height: 12),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerLow,
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(
                _statusMessage!,
                style: theme.textTheme.bodySmall,
              ),
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _retryExecution(String executionId) async {
    setState(() {
      _busy = true;
      _statusMessage = null;
    });
    try {
      final notifier = ref.read(trustageActionProvider.notifier);
      final exec = await notifier.retryExecution(executionId);
      if (mounted) {
        setState(() {
          _busy = false;
          _statusMessage = 'Retry scheduled for execution ${shortId(exec.id)}';
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

  Future<void> _retryInstance(String instanceId) async {
    setState(() {
      _busy = true;
      _statusMessage = null;
    });
    try {
      final notifier = ref.read(trustageActionProvider.notifier);
      await notifier.retryInstance(instanceId);
      if (mounted) {
        setState(() {
          _busy = false;
          _statusMessage = 'Retry scheduled for instance ${shortId(instanceId)}';
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

  Future<void> _sendSignal() async {
    if (_signalNameCtl.text.isEmpty) {
      setState(() => _statusMessage = 'Signal name is required');
      return;
    }

    Map<String, dynamic>? payload;
    try {
      payload = jsonDecode(_signalPayloadCtl.text) as Map<String, dynamic>;
    } catch (e) {
      setState(() => _statusMessage = 'Invalid JSON: $e');
      return;
    }

    setState(() {
      _busy = true;
      _statusMessage = null;
    });

    try {
      final notifier = ref.read(trustageActionProvider.notifier);
      final delivered = await notifier.sendSignal(
        widget.instance.id,
        _signalNameCtl.text,
        payload,
      );
      if (mounted) {
        setState(() {
          _busy = false;
          _statusMessage = delivered
              ? 'Signal "${_signalNameCtl.text}" delivered'
              : 'Signal "${_signalNameCtl.text}" queued';
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
