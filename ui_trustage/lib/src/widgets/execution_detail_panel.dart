import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import 'package:antinvestor_api_runtime/antinvestor_api_runtime.dart' as rt;

import 'json_block.dart';
import 'status_helpers.dart';
import 'trustage_panel.dart';
import 'trustage_status_badge.dart';

/// Detail panel for a selected execution showing input/output payloads.
class ExecutionDetailPanel extends StatelessWidget {
  const ExecutionDetailPanel({
    super.key,
    required this.execution,
    this.output,
  });

  final rt.WorkflowExecution execution;
  final rt.StateOutput? output;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final fmt = DateFormat.yMMMd().add_jm();

    return TrustagePanel(
      eyebrow: 'Execution',
      title: execution.state,
      subtitle: shortId(execution.id),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              TrustageStatusBadge(status: execution.status.name),
              const SizedBox(width: 8),
              Text(
                'Attempt #${execution.attempt}',
                style: theme.textTheme.labelSmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),

          // Metadata
          if (execution.traceId.isNotEmpty)
            _DetailRow(label: 'Trace ID', value: execution.traceId),
          if (execution.errorClass.isNotEmpty)
            _DetailRow(label: 'Error Class', value: execution.errorClass),
          if (execution.hasStartedAt())
            _DetailRow(
              label: 'Started',
              value: fmt.format(execution.startedAt.toDateTime()),
            ),
          if (execution.hasFinishedAt())
            _DetailRow(
              label: 'Finished',
              value: fmt.format(execution.finishedAt.toDateTime()),
            ),
          if (execution.inputSchemaHash.isNotEmpty)
            _DetailRow(
              label: 'Input Schema',
              value: shortId(execution.inputSchemaHash),
            ),
          if (execution.outputSchemaHash.isNotEmpty)
            _DetailRow(
              label: 'Output Schema',
              value: shortId(execution.outputSchemaHash),
            ),

          // Error message
          if (execution.errorMessage.isNotEmpty) ...[
            const SizedBox(height: 12),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: const Color(0xFFFFF1F2),
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: const Color(0xFFFECDD3)),
              ),
              child: Text(
                execution.errorMessage,
                style: const TextStyle(
                    fontSize: 12, color: Color(0xFFBE123C)),
              ),
            ),
          ],

          // Input payload
          if (execution.hasInputPayload()) ...[
            const SizedBox(height: 16),
            JsonBlock(
              label: 'Input Payload',
              value: _structToMap(execution.inputPayload),
            ),
          ],

          // Output payload
          if (output != null && output!.hasPayload()) ...[
            const SizedBox(height: 12),
            JsonBlock(
              label: 'Output Payload',
              value: _structToMap(output!.payload),
            ),
          ] else if (execution.hasOutput()) ...[
            const SizedBox(height: 12),
            JsonBlock(
              label: 'Output',
              value: _structToMap(execution.output),
            ),
          ],
        ],
      ),
    );
  }

  Map<String, dynamic> _structToMap(dynamic s) {
    if (s == null) return {};
    try {
      final fields = (s as dynamic).fields as Map;
      return fields.map((k, v) => MapEntry(k.toString(), _valueToNative(v)));
    } catch (_) {
      return {};
    }
  }

  dynamic _valueToNative(dynamic v) {
    if (v == null) return null;
    try {
      if (v.hasStringValue()) return v.stringValue;
      if (v.hasNumberValue()) return v.numberValue;
      if (v.hasBoolValue()) return v.boolValue;
      if (v.hasStructValue()) return _structToMap(v.structValue);
      if (v.hasListValue()) {
        return (v.listValue.values as List).map(_valueToNative).toList();
      }
    } catch (_) {}
    return v.toString();
  }
}

class _DetailRow extends StatelessWidget {
  const _DetailRow({required this.label, required this.value});
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 4),
      child: Row(
        children: [
          SizedBox(
            width: 110,
            child: Text(
              label,
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ),
          Expanded(
            child: SelectableText(
              value,
              style: theme.textTheme.bodySmall?.copyWith(
                fontFamily: 'monospace',
                fontSize: 12,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
